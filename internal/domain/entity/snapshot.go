package entity

import "time"

// Snapshot is a serialised point-in-time copy of the board.
type Snapshot struct {
	ID      int64
	Name    string
	Data    string    // JSON blob
	Hash    string    // SHA-256 of Data for deduplication
	SavedAt time.Time
}
