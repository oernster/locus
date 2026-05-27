package entity

import "time"

// Command is a task managed on the board.
type Command struct {
	ID        int64
	Title     string
	Status    Status
	StageId   StageId
	SortIndex int
	CreatedAt time.Time
}
