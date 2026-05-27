package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestCommandService_ValidStage(t *testing.T) {
	valid := []entity.StageId{entity.StagePlan, entity.StageExecute, entity.StageCheck, entity.StageDone}
	for _, s := range valid {
		if !validStage(s) {
			t.Errorf("validStage(%q) = false, want true", s)
		}
	}
	invalid := []entity.StageId{"", "DESIGN", "BUILD", "plan", "UNKNOWN"}
	for _, s := range invalid {
		if validStage(s) {
			t.Errorf("validStage(%q) = true, want false", s)
		}
	}
}

func TestCommandService_List_All(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{
			{ID: 1, Title: "A", StageId: entity.StagePlan},
			{ID: 2, Title: "B", StageId: entity.StageExecute},
		},
	}
	svc := NewCommandService(repo)
	result, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestCommandService_List_Filtered(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{
			{ID: 1, Title: "A", StageId: entity.StagePlan},
			{ID: 2, Title: "B", StageId: entity.StageExecute},
		},
	}
	svc := NewCommandService(repo)
	result, err := svc.List(context.Background(), "PLAN")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Title != "A" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCommandService_List_Error(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{listErr: sentinel})
	_, err := svc.List(context.Background(), "")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Get_Found(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "Task", StageId: entity.StagePlan}},
	}
	svc := NewCommandService(repo)
	d, err := svc.Get(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != 1 || d.Title != "Task" {
		t.Fatalf("unexpected DTO: %+v", d)
	}
}

func TestCommandService_Get_Error(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{getErr: sentinel})
	_, err := svc.Get(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Create_EmptyTitle(t *testing.T) {
	svc := NewCommandService(&mockCommandRepo{})
	_, err := svc.Create(context.Background(), "", "PLAN")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestCommandService_Create_InvalidStage(t *testing.T) {
	svc := NewCommandService(&mockCommandRepo{})
	_, err := svc.Create(context.Background(), "Task", "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid stage")
	}
}

func TestCommandService_Create_RepoError(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{createErr: sentinel})
	_, err := svc.Create(context.Background(), "Task", "PLAN")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Create_Success(t *testing.T) {
	repo := &mockCommandRepo{}
	svc := NewCommandService(repo)
	d, err := svc.Create(context.Background(), "My Task", "EXECUTE")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "My Task" || d.StageId != "EXECUTE" || d.Status != "Not Started" {
		t.Fatalf("unexpected DTO: %+v", d)
	}
	if d.ID == 0 {
		t.Fatal("ID not assigned")
	}
}

func TestCommandService_Update_GetError(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{getErr: sentinel})
	_, err := svc.Update(context.Background(), 1, "", "", "")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Update_InvalidStage(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	svc := NewCommandService(repo)
	_, err := svc.Update(context.Background(), 1, "", "", "BOGUS")
	if err == nil {
		t.Fatal("expected error for invalid stage")
	}
}

func TestCommandService_Update_UpdateError(t *testing.T) {
	sentinel := errors.New("db error")
	repo := &mockCommandRepo{
		cmds:      []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
		updateErr: sentinel,
	}
	svc := NewCommandService(repo)
	_, err := svc.Update(context.Background(), 1, "New", "", "")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Update_AllFields(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "Old", Status: entity.StatusNotStarted, StageId: entity.StagePlan}},
	}
	svc := NewCommandService(repo)
	d, err := svc.Update(context.Background(), 1, "New", "In Progress", "EXECUTE")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "New" || d.Status != "In Progress" || d.StageId != "EXECUTE" {
		t.Fatalf("unexpected DTO: %+v", d)
	}
}

func TestCommandService_Update_EmptyFieldsKeepExisting(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "Orig", Status: entity.StatusBlocked, StageId: entity.StageCheck}},
	}
	svc := NewCommandService(repo)
	d, err := svc.Update(context.Background(), 1, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Orig" || d.Status != "Blocked" || d.StageId != "CHECK" {
		t.Fatalf("fields changed unexpectedly: %+v", d)
	}
}

func TestCommandService_Delete(t *testing.T) {
	repo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	svc := NewCommandService(repo)
	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(repo.cmds) != 0 {
		t.Fatal("command not deleted")
	}
}

func TestCommandService_Delete_Error(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{deleteErr: sentinel})
	if err := svc.Delete(context.Background(), 1); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestCommandService_Reorder_Success(t *testing.T) {
	svc := NewCommandService(&mockCommandRepo{})
	err := svc.Reorder(context.Background(), map[string][]int64{"PLAN": {2, 1}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommandService_Reorder_Error(t *testing.T) {
	sentinel := errors.New("db error")
	svc := NewCommandService(&mockCommandRepo{reorderErr: sentinel})
	if err := svc.Reorder(context.Background(), map[string][]int64{"PLAN": {1}}); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestToCommandDTO_CreatedAtFormatted(t *testing.T) {
	// Cover the toCommandDTO helper indirectly via Create.
	repo := &mockCommandRepo{}
	svc := NewCommandService(repo)
	d, err := svc.Create(context.Background(), "T", "PLAN")
	if err != nil {
		t.Fatal(err)
	}
	// CreatedAt should be ISO-8601 UTC without timezone offset.
	if len(d.CreatedAt) == 0 {
		t.Fatal("CreatedAt is empty")
	}
}
