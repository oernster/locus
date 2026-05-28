package entity

import "time"

// Source identifies the origin of a Command.
const (
	SourceManual = "manual"
	SourceClaude = "claude"
)

// Command is a task managed on the board.
type Command struct {
	ID         int64
	Title      string
	Status     Status
	StageId    StageId
	SortIndex  int
	CreatedAt  time.Time
	Source     string     // SourceManual or SourceClaude
	SessionID  string     // Claude session UUID; empty for manual commands
	ArchivedAt *time.Time // non-nil when this dynamic item has been archived
}
