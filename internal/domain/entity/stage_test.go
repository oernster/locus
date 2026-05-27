package entity

import "testing"

func TestStageIdConstants(t *testing.T) {
	if StagePlan != "PLAN" {
		t.Errorf("StagePlan = %q, want PLAN", StagePlan)
	}
	if StageExecute != "EXECUTE" {
		t.Errorf("StageExecute = %q, want EXECUTE", StageExecute)
	}
	if StageCheck != "CHECK" {
		t.Errorf("StageCheck = %q, want CHECK", StageCheck)
	}
	if StageDone != "DONE" {
		t.Errorf("StageDone = %q, want DONE", StageDone)
	}
}

func TestStagesOrder(t *testing.T) {
	want := []StageId{StagePlan, StageExecute, StageCheck, StageDone}
	if len(Stages) != len(want) {
		t.Fatalf("len(Stages) = %d, want %d", len(Stages), len(want))
	}
	for i, s := range want {
		if Stages[i] != s {
			t.Errorf("Stages[%d] = %q, want %q", i, Stages[i], s)
		}
	}
}

func TestStatusConstants(t *testing.T) {
	cases := map[Status]string{
		StatusNotStarted: "Not Started",
		StatusInProgress: "In Progress",
		StatusBlocked:    "Blocked",
		StatusComplete:   "Complete",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("Status = %q, want %q", got, want)
		}
	}
}
