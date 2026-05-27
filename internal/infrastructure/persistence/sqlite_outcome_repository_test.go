package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestSQLiteOutcomeRepository_CreateAndList(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteOutcomeRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)

	o := entity.Outcome{CommandID: cmd.ID, Note: "done", CreatedAt: time.Now().UTC()}
	created, err := repo.Create(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("ID not assigned")
	}

	list, err := repo.ListByCommandID(ctx, cmd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Note != "done" {
		t.Fatalf("unexpected list: %v", list)
	}
}

func TestSQLiteOutcomeRepository_Create_ZeroCreatedAt(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteOutcomeRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	o := entity.Outcome{CommandID: cmd.ID, Note: "note"}
	created, err := repo.Create(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be defaulted")
	}
}

func TestSQLiteOutcomeRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteOutcomeRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	created, _ := repo.Create(ctx, entity.Outcome{CommandID: cmd.ID, Note: "x", CreatedAt: time.Now().UTC()})

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := repo.ListByCommandID(ctx, cmd.ID)
	if len(list) != 0 {
		t.Fatal("outcome should be deleted")
	}
}

func TestSQLiteOutcomeRepository_ListByCommandID_Empty(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteOutcomeRepository(db)
	list, err := repo.ListByCommandID(context.Background(), 9999)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty, got %v", list)
	}
}

func TestSQLiteOutcomeRepository_List_OrderedByCreatedAtDesc(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteOutcomeRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	now := time.Now().UTC()
	_, _ = repo.Create(ctx, entity.Outcome{CommandID: cmd.ID, Note: "first", CreatedAt: now})
	_, _ = repo.Create(ctx, entity.Outcome{CommandID: cmd.ID, Note: "second", CreatedAt: now.Add(time.Second)})

	list, err := repo.ListByCommandID(ctx, cmd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	// DESC order: second created last comes first.
	if list[0].Note != "second" {
		t.Fatalf("expected DESC order, got %v", list)
	}
}
