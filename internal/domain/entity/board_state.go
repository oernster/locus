package entity

// BoardState is the singleton that holds board-level metadata.
type BoardState struct {
	Name        string
	UserNamed   bool
	StageLabels map[string]string // label overrides per stage; nil means all defaults
}
