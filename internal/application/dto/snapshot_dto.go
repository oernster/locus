package dto

// SnapshotDTO is the data transfer object for a Snapshot summary.
type SnapshotDTO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	SavedAt string `json:"saved_at"` // ISO 8601 UTC
}
