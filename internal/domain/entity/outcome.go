package entity

import "time"

// Outcome records a note about the result of work on a Command.
type Outcome struct {
	ID        int64
	CommandID int64
	Note      string
	CreatedAt time.Time
}
