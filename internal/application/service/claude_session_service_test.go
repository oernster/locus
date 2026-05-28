package service

import (
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// newClaudeSvc builds a ClaudeSessionService backed by the supplied mock.
func newClaudeSvc(repo *mockCommandRepo) (*ClaudeSessionService, *int) {
	notifyCalls := 0
	svc := NewClaudeSessionService(repo, nil, func() { notifyCalls++ })
	return svc, &notifyCalls
}

// makeEvent builds a ClaudeEvent with required fields.
func makeEvent(typ entity.ClaudeEventType, sessionID, tool, target string, success bool) entity.ClaudeEvent {
	return entity.ClaudeEvent{
		Type:      typ,
		SessionID: sessionID,
		Tool:      tool,
		Target:    target,
		Success:   success,
		Timestamp: time.Now().Unix(),
	}
}

// --- inferStage tests ---

func TestInferStage_EditWrite(t *testing.T) {
	for _, tool := range []string{"Edit", "Write", "NotebookEdit"} {
		if got := inferStage(tool, ""); got != entity.StageExecute {
			t.Errorf("inferStage(%q) = %v, want EXECUTE", tool, got)
		}
	}
}

func TestInferStage_Bash_TestKeyword(t *testing.T) {
	cases := []string{"go test ./...", "npm test", "pytest tests/", "jest --watch"}
	for _, cmd := range cases {
		if got := inferStage("Bash", cmd); got != entity.StageCheck {
			t.Errorf("inferStage(Bash, %q) = %v, want CHECK", cmd, got)
		}
	}
}

func TestInferStage_Bash_NonTest(t *testing.T) {
	if got := inferStage("Bash", "ls -la"); got != entity.StageExecute {
		t.Errorf("inferStage(Bash, ls) = %v, want EXECUTE", got)
	}
}

func TestInferStage_Unknown(t *testing.T) {
	if got := inferStage("Read", "file.go"); got != entity.StagePlan {
		t.Errorf("inferStage(Read) = %v, want PLAN", got)
	}
}


// --- formatTitle tests ---

func TestFormatTitle_NoTarget(t *testing.T) {
	if got := formatTitle("Edit", ""); got != "Edit" {
		t.Errorf("formatTitle = %q, want %q", got, "Edit")
	}
}

func TestFormatTitle_WithFile(t *testing.T) {
	got := formatTitle("Edit", "internal/auth/middleware.go")
	want := "Edit: middleware.go"
	if got != want {
		t.Errorf("formatTitle = %q, want %q", got, want)
	}
}

func TestFormatTitle_LongTarget(t *testing.T) {
	long := "averylongtargetnamethatiswellover60characters_yes_really_yes.go"
	got := formatTitle("Bash", long)
	if len(got) > 70 { // "Bash: " + 63
		t.Errorf("formatTitle too long: %d chars", len(got))
	}
}

// --- HandleEvent: tool_use creates item ---

func TestHandleEvent_ToolUse_CreatesItem(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "main.go", false))

	if len(repo.cmds) != 1 {
		t.Fatalf("want 1 cmd, got %d", len(repo.cmds))
	}
	cmd := repo.cmds[0]
	if cmd.Source != entity.SourceClaude {
		t.Errorf("Source = %q, want %q", cmd.Source, entity.SourceClaude)
	}
	if cmd.SessionID != "s1" {
		t.Errorf("SessionID = %q, want %q", cmd.SessionID, "s1")
	}
	if cmd.StageId != entity.StageExecute {
		t.Errorf("StageId = %v, want EXECUTE", cmd.StageId)
	}
	if *notify != 1 {
		t.Errorf("notifyCalls = %d, want 1", *notify)
	}
}

func TestHandleEvent_ToolUse_NonItemTool_NoCreate(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Read", "main.go", false))

	if len(repo.cmds) != 0 {
		t.Errorf("want 0 cmds, got %d", len(repo.cmds))
	}
	if *notify != 0 {
		t.Errorf("notifyCalls = %d, want 0", *notify)
	}
}

func TestHandleEvent_ToolUse_CreateError_NoNotify(t *testing.T) {
	repo := &mockCommandRepo{createErr: errNotFound}
	svc, notify := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "x.go", false))

	if *notify != 0 {
		t.Errorf("should not notify on create error")
	}
}

// --- HandleEvent: tool_result advances stage ---

func TestHandleEvent_ToolResult_AdvancesStage(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	// Create item via tool_use.
	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "x.go", false))
	beforeNotify := *notify

	// Advance via successful tool_result.
	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Edit", "x.go", true))

	if len(repo.cmds) != 1 {
		t.Fatalf("want 1 cmd, got %d", len(repo.cmds))
	}
	if repo.cmds[0].StageId != entity.StageDone {
		t.Errorf("StageId = %v, want DONE", repo.cmds[0].StageId)
	}
	if repo.cmds[0].Status != entity.StatusComplete {
		t.Errorf("Status = %v, want Complete", repo.cmds[0].Status)
	}
	if *notify <= beforeNotify {
		t.Error("should notify on advance")
	}
}

func TestHandleEvent_ToolResult_Failure_NoAdvance(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, _ := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "x.go", false))
	origStage := repo.cmds[0].StageId

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Edit", "x.go", false))

	if repo.cmds[0].StageId != origStage {
		t.Errorf("stage changed on failure: %v -> %v", origStage, repo.cmds[0].StageId)
	}
}

func TestHandleEvent_ToolResult_NoPending_NoOp(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	// No prior tool_use; result should be a no-op.
	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Edit", "x.go", true))

	if len(repo.cmds) != 0 {
		t.Errorf("want 0 cmds, got %d", len(repo.cmds))
	}
	if *notify != 0 {
		t.Errorf("notifyCalls = %d, want 0", *notify)
	}
}

func TestHandleEvent_ToolResult_NonItemTool_NoOp(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Read", "x.go", true))

	if *notify != 0 {
		t.Errorf("should not notify for non-item tool result")
	}
}

// --- HandleEvent: CHECK->DONE on Bash success ---

func TestHandleEvent_BashTest_AdvancesToDone(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, _ := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Bash", "go test ./...", false))
	if repo.cmds[0].StageId != entity.StageCheck {
		t.Fatalf("Bash test should start at CHECK, got %v", repo.cmds[0].StageId)
	}

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Bash", "go test ./...", true))

	if repo.cmds[0].StageId != entity.StageDone {
		t.Errorf("StageId = %v, want DONE", repo.cmds[0].StageId)
	}
	if repo.cmds[0].Status != entity.StatusComplete {
		t.Errorf("Status = %v, want Complete", repo.cmds[0].Status)
	}
}

// --- HandleEvent: session_end archives ---

func TestHandleEvent_SessionEnd_PreservesItems(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, notify := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "x.go", false))
	beforeNotify := *notify

	svc.HandleEvent(entity.ClaudeEvent{
		Type:      entity.ClaudeEventSessionEnd,
		SessionID: "s1",
		Timestamp: time.Now().Unix(),
	})

	// Items must remain on the board after session end; user clears manually.
	if len(repo.cmds) != 1 {
		t.Errorf("want 1 cmd preserved after session end, got %d", len(repo.cmds))
	}
	if *notify <= beforeNotify {
		t.Error("should notify on session end")
	}
}

// --- Multiple sessions are independent ---

func TestHandleEvent_MultipleSessions_Independent(t *testing.T) {
	repo := &mockCommandRepo{}
	svc, _ := newClaudeSvc(repo)

	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s1", "Edit", "a.go", false))
	svc.HandleEvent(makeEvent(entity.ClaudeEventToolUse, "s2", "Edit", "b.go", false))

	if len(repo.cmds) != 2 {
		t.Fatalf("want 2 cmds, got %d", len(repo.cmds))
	}

	// Advance s1 only.
	svc.HandleEvent(makeEvent(entity.ClaudeEventToolResult, "s1", "Edit", "a.go", true))

	s1Cmd := repo.cmds[0]
	s2Cmd := repo.cmds[1]
	if s1Cmd.SessionID == "s1" && s1Cmd.StageId != entity.StageDone {
		t.Errorf("s1 cmd: StageId = %v, want DONE", s1Cmd.StageId)
	}
	if s2Cmd.SessionID == "s2" && s2Cmd.StageId != entity.StageExecute {
		t.Errorf("s2 cmd unexpectedly advanced: StageId = %v", s2Cmd.StageId)
	}
}
