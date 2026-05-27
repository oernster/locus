package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestOutcomeService_ListByCommand_Empty(t *testing.T) {
	svc := NewOutcomeService(&mockOutcomeRepo{})
	result, err := svc.ListByCommand(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestOutcomeService_ListByCommand_Populated(t *testing.T) {
	repo := &mockOutcomeRepo{
		outcomes: []entity.Outcome{
			{ID: 1, CommandID: 42, Note: "done"},
			{ID: 2, CommandID: 99, Note: "other"},
		},
	}
	svc := NewOutcomeService(repo)
	result, err := svc.ListByCommand(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Note != "done" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestOutcomeService_ListByCommand_Error(t *testing.T) {
	sentinel := errors.New("db err")
	svc := NewOutcomeService(&mockOutcomeRepo{listErr: sentinel})
	_, err := svc.ListByCommand(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestOutcomeService_Create_EmptyNote(t *testing.T) {
	svc := NewOutcomeService(&mockOutcomeRepo{})
	_, err := svc.Create(context.Background(), 1, "")
	if err == nil {
		t.Fatal("expected error for empty note")
	}
}

func TestOutcomeService_Create_RepoError(t *testing.T) {
	sentinel := errors.New("create err")
	svc := NewOutcomeService(&mockOutcomeRepo{createErr: sentinel})
	_, err := svc.Create(context.Background(), 1, "note")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestOutcomeService_Create_Success(t *testing.T) {
	repo := &mockOutcomeRepo{}
	svc := NewOutcomeService(repo)
	d, err := svc.Create(context.Background(), 7, "finished")
	if err != nil {
		t.Fatal(err)
	}
	if d.Note != "finished" || d.CommandID != 7 || d.ID == 0 {
		t.Fatalf("unexpected DTO: %+v", d)
	}
	if d.CreatedAt == "" {
		t.Fatal("CreatedAt should be set")
	}
}

func TestOutcomeService_Delete_Success(t *testing.T) {
	repo := &mockOutcomeRepo{
		outcomes: []entity.Outcome{{ID: 3, CommandID: 1, Note: "n"}},
	}
	svc := NewOutcomeService(repo)
	if err := svc.Delete(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	if len(repo.outcomes) != 0 {
		t.Fatal("outcome should be deleted")
	}
}

func TestOutcomeService_Delete_Error(t *testing.T) {
	sentinel := errors.New("delete err")
	svc := NewOutcomeService(&mockOutcomeRepo{deleteErr: sentinel})
	if err := svc.Delete(context.Background(), 1); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
