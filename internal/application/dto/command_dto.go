package dto

// CommandDTO is the data transfer object for a Command.
type CommandDTO struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	StageId   string `json:"stage_id"`
	SortIndex int    `json:"sort_index"`
	CreatedAt string `json:"created_at"` // ISO 8601 UTC
}
