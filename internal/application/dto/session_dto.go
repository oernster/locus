package dto

// SessionDTO is the data transfer object for a Session.
type SessionDTO struct {
	Active    bool   `json:"active"`
	ID        int64  `json:"id,omitempty"`
	CommandID int64  `json:"command_id,omitempty"`
	StageId   string `json:"stage_id,omitempty"`
	StartedAt string `json:"started_at,omitempty"` // ISO 8601 UTC
	EndedAt   string `json:"ended_at,omitempty"`  // ISO 8601 UTC
}
