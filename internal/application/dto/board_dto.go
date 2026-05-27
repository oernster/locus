package dto

// BoardDTO is the data transfer object for the board state.
type BoardDTO struct {
	Name         string            `json:"name"`
	UserNamed    bool              `json:"user_named"`
	IsNewUnnamed bool              `json:"is_new_unnamed"`
	IsEmpty      bool              `json:"is_empty"`
	StageLabels  map[string]string `json:"stage_labels,omitempty"`
}
