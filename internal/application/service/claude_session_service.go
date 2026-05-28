package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// itemTools lists the tool names whose invocations produce a visible board item.
// Research tools (Read, Glob, Grep, WebSearch, WebFetch) are excluded to avoid noise.
var itemTools = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"NotebookEdit": true,
	"Bash":         true,
}

// testKeywords are substrings that, when present in a Bash command, indicate a
// CHECK-stage action (running tests or linting).
var testKeywords = []string{
	"go test", "npm test", "yarn test", "pnpm test",
	"pytest", "jest", "vitest", "mocha", "cargo test",
	"dotnet test", "mvn test", "gradle test",
}

// trivialBashPrefixes are command prefixes that produce no meaningful board item.
// These are read-only, navigational, or output-only operations that don't
// represent conceptual work worth recording.
var trivialBashPrefixes = []string{
	"cd ", "ls ", "ls\t", "dir ", "pwd",
	"echo ", "printf ",
	"grep ", "rg ", "find ", "locate ",
	"cat ", "head ", "tail ", "less ", "more ",
	"sed ", "awk ", "wc ", "sort ", "uniq ", "cut ", "tr ",
	"which ", "where ", "type ",
	"date", "env ", "set ",
	// PowerShell read-only equivalents
	"get-childitem", "select-string", "write-output", "write-host",
	"get-content", "measure-object", "get-location", "test-path",
}

// meaningfulSegment strips leading trivial segments (e.g. "cd /path && ") from
// a compound command and returns the first meaningful remainder. If the whole
// command is trivial, it returns the original string.
func meaningfulSegment(cmd string) string {
	remaining := strings.TrimSpace(cmd)
	for {
		// Split off one segment at the first && or ;
		var head, tail string
		if i := strings.Index(remaining, " && "); i >= 0 {
			head, tail = remaining[:i], strings.TrimSpace(remaining[i+4:])
		} else if i := strings.Index(remaining, "; "); i >= 0 {
			head, tail = remaining[:i], strings.TrimSpace(remaining[i+2:])
		} else {
			break
		}
		if isTrivialSegment(head) && tail != "" {
			remaining = tail
		} else {
			break
		}
	}
	return remaining
}

// isTrivialSegment checks a single (non-compound) command for triviality.
func isTrivialSegment(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" || trimmed == "null" {
		return true
	}
	lc := strings.ToLower(trimmed)
	for _, prefix := range trivialBashPrefixes {
		bare := strings.TrimRight(prefix, " \t")
		if lc == bare || strings.HasPrefix(lc, prefix) {
			return true
		}
	}
	return false
}

// isTrivialBash returns true for commands that represent no meaningful
// conceptual work and should not appear on the board.
func isTrivialBash(cmd string) bool {
	return isTrivialSegment(meaningfulSegment(cmd))
}

// ClaudeSessionService processes ClaudeEvents emitted by Claude Code hooks and
// creates, advances, and archives dynamic board items accordingly.
type ClaudeSessionService struct {
	commandRepo repository.CommandRepository
	snapshotSvc *SnapshotService
	notifyFn    func()

	mu sync.Mutex
	// pending maps session_id to an ordered stack of command IDs created for
	// item-creating tool calls. tool_result events pop from this stack to
	// identify which item to advance.
	pending map[string][]int64
}

// NewClaudeSessionService creates the service.
// snapshotSvc may be nil. notifyFn is called after each board-mutating event (may be nil).
func NewClaudeSessionService(
	commandRepo repository.CommandRepository,
	snapshotSvc *SnapshotService,
	notifyFn func(),
) *ClaudeSessionService {
	return &ClaudeSessionService{
		commandRepo: commandRepo,
		snapshotSvc: snapshotSvc,
		notifyFn:    notifyFn,
		pending:     make(map[string][]int64),
	}
}

// HandleEvent processes a single ClaudeEvent. It is safe to call concurrently.
func (s *ClaudeSessionService) HandleEvent(ev entity.ClaudeEvent) {
	ctx := context.Background()
	switch ev.Type {
	case entity.ClaudeEventToolUse:
		s.handleToolUse(ctx, ev)
	case entity.ClaudeEventToolResult:
		s.handleToolResult(ctx, ev)
	case entity.ClaudeEventSessionEnd:
		s.handleSessionEnd(ctx, ev)
	}
}

func (s *ClaudeSessionService) handleToolUse(ctx context.Context, ev entity.ClaudeEvent) {
	if !itemTools[ev.Tool] {
		return
	}
	if ev.Tool == "Bash" && isTrivialBash(ev.Target) {
		return
	}
	ts := time.Unix(ev.Timestamp, 0).UTC()
	if ev.Timestamp == 0 {
		ts = time.Now().UTC()
	}
	cmd := entity.Command{
		Title:     formatTitle(ev.Tool, ev.Target),
		Status:    entity.StatusNotStarted,
		StageId:   inferStage(ev.Tool, ev.Target),
		Source:    entity.SourceClaude,
		SessionID: ev.SessionID,
		CreatedAt: ts,
	}
	created, err := s.commandRepo.Create(ctx, cmd)
	if err != nil {
		log.Printf("claude_session: create item: %v", err)
		return
	}

	s.mu.Lock()
	s.pending[ev.SessionID] = append(s.pending[ev.SessionID], created.ID)
	s.mu.Unlock()

	s.notify()
}

func (s *ClaudeSessionService) handleToolResult(ctx context.Context, ev entity.ClaudeEvent) {
	if !itemTools[ev.Tool] {
		return
	}

	s.mu.Lock()
	ids := s.pending[ev.SessionID]
	if len(ids) == 0 {
		s.mu.Unlock()
		return
	}
	id := ids[len(ids)-1]
	s.pending[ev.SessionID] = ids[:len(ids)-1]
	s.mu.Unlock()

	if !ev.Success {
		return // Item stays at current stage on failure.
	}

	cmd, err := s.commandRepo.Get(ctx, id)
	if err != nil {
		log.Printf("claude_session: get item %d: %v", id, err)
		return
	}
	if cmd.StageId == entity.StageDone {
		return // Already at DONE.
	}
	cmd.StageId = entity.StageDone
	cmd.Status = entity.StatusComplete
	if err := s.commandRepo.Update(ctx, cmd); err != nil {
		log.Printf("claude_session: advance stage for item %d: %v", id, err)
		return
	}

	s.notify()
}

func (s *ClaudeSessionService) handleSessionEnd(ctx context.Context, ev entity.ClaudeEvent) {
	s.mu.Lock()
	delete(s.pending, ev.SessionID)
	s.mu.Unlock()

	if s.snapshotSvc != nil {
		cmds, err := s.commandRepo.List(ctx, nil)
		if err == nil && len(cmds) > 0 {
			name := fmt.Sprintf("Session %s", time.Now().UTC().Format("2006-01-02 15:04"))
			if _, serr := s.snapshotSvc.Save(ctx, name); serr != nil {
				log.Printf("claude_session: auto-snapshot: %v", serr)
			}
		}
	}

	s.notify()
}

func (s *ClaudeSessionService) notify() {
	if s.notifyFn != nil {
		s.notifyFn()
	}
}

// inferStage maps a tool name (and optional target) to the appropriate board stage.
func inferStage(tool, target string) entity.StageId {
	switch tool {
	case "Edit", "Write", "NotebookEdit":
		return entity.StageExecute
	case "Bash":
		lc := strings.ToLower(target)
		for _, kw := range testKeywords {
			if strings.Contains(lc, kw) {
				return entity.StageCheck
			}
		}
		return entity.StageExecute
	default:
		return entity.StagePlan
	}
}

// formatTitle builds a short human-readable label for a dynamic board item.
func formatTitle(tool, target string) string {
	switch tool {
	case "Edit", "Write", "NotebookEdit":
		return fileTitle(tool, target)
	case "Bash":
		return bashTitle(target)
	default:
		return tool
	}
}

func fileTitle(tool, target string) string {
	if target == "" {
		return tool
	}
	parts := strings.FieldsFunc(target, func(r rune) bool { return r == '/' || r == '\\' })
	name := target
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}
	if len(name) > 50 {
		name = name[:47] + "..."
	}
	return fmt.Sprintf("%s: %s", tool, name)
}

func bashTitle(cmd string) string {
	if strings.TrimSpace(cmd) == "" || cmd == "null" {
		return "Shell: command"
	}
	cmd = meaningfulSegment(cmd)
	lc := strings.ToLower(strings.TrimSpace(cmd))

	// git operations
	if strings.HasPrefix(lc, "git ") {
		tokens := strings.Fields(lc)
		if len(tokens) >= 2 {
			return "Git: " + tokens[1]
		}
		return "Git: operation"
	}

	// test runners
	for _, kw := range testKeywords {
		if strings.Contains(lc, kw) {
			return "Test: run suite"
		}
	}

	// build
	for _, kw := range []string{"go build", "wails build", "npm run build", "cargo build", "dotnet build", "mvn package", "gradle build", "make"} {
		if strings.Contains(lc, kw) {
			return "Build: compile"
		}
	}

	// dependency management
	for _, kw := range []string{"npm install", "npm i ", "yarn install", "pnpm install", "go get", "go mod", "pip install", "cargo add"} {
		if strings.Contains(lc, kw) {
			return "Install: dependencies"
		}
	}

	// launch / run app
	for _, kw := range []string{"start-process", "wails dev", ".exe", ".sh", ".py", "node ", "go run"} {
		if strings.Contains(lc, kw) {
			return "Launch: app"
		}
	}

	// file operations
	for _, kw := range []string{"cp ", "mv ", "rm ", "mkdir", "copy ", "move ", "del ", "remove-item", "new-item", "copy-item", "move-item"} {
		if strings.Contains(lc, kw) {
			return "Files: organise"
		}
	}

	// fallback: show enough of the actual command to understand what it does
	trimmed := strings.TrimSpace(cmd)
	if len(trimmed) > 64 {
		return trimmed[:61] + "..."
	}
	return trimmed
}
