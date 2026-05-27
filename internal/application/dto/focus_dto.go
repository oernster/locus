package dto

// FocusDataDTO contains aggregated focus-reader data for a board stage.
type FocusDataDTO struct {
	Available       bool          `json:"available"`
	StageId         string        `json:"stage_id"`
	TotalSeconds    int64         `json:"total_seconds"`
	IdleSeconds     int64         `json:"idle_seconds"`
	DeepWorkSeconds int64         `json:"deep_work_seconds"`
	Apps            []AppFocusDTO `json:"apps"`
}

// AppFocusDTO holds per-application focus statistics.
type AppFocusDTO struct {
	ExePath      string `json:"exe_path"`
	FriendlyName string `json:"friendly_name"`
	TotalSeconds int64  `json:"total_seconds"`
	SessionCount int    `json:"session_count"`
}
