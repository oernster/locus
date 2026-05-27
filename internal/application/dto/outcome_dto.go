package dto

// OutcomeDTO is the data transfer object for an Outcome.
type OutcomeDTO struct {
	ID        int64  `json:"id"`
	CommandID int64  `json:"command_id"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"` // ISO 8601 UTC
}
