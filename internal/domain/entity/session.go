package entity

import "time"

// Session tracks time spent on a Command within a Stage.
type Session struct {
	ID        int64
	CommandID int64
	StageId   StageId
	StartedAt time.Time
	EndedAt   *time.Time
}
