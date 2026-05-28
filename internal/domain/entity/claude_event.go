package entity

// ClaudeEventType identifies the kind of event emitted by a Claude Code hook.
type ClaudeEventType string

const (
	ClaudeEventSessionStart ClaudeEventType = "session_start"
	ClaudeEventToolUse      ClaudeEventType = "tool_use"
	ClaudeEventToolResult   ClaudeEventType = "tool_result"
	ClaudeEventSessionEnd   ClaudeEventType = "session_end"
)

// ClaudeEvent is one line in the hook JSONL sidecar file written by Claude Code hooks.
type ClaudeEvent struct {
	Type      ClaudeEventType `json:"type"`
	SessionID string          `json:"session_id"`
	Tool      string          `json:"tool,omitempty"`
	Target    string          `json:"target,omitempty"`
	Success   bool            `json:"success,omitempty"`
	Timestamp int64           `json:"ts"`
}
