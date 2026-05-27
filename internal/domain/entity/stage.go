package entity

// StageId identifies a board stage.
type StageId string

const (
	StagePlan    StageId = "PLAN"
	StageExecute StageId = "EXECUTE"
	StageCheck   StageId = "CHECK"
	StageDone    StageId = "DONE"
)

// Stages lists all stage IDs in display order.
var Stages = []StageId{StagePlan, StageExecute, StageCheck, StageDone}

// Status represents the workflow state of a Command.
type Status string

const (
	StatusNotStarted Status = "Not Started"
	StatusInProgress Status = "In Progress"
	StatusBlocked    Status = "Blocked"
	StatusComplete   Status = "Complete"
)
