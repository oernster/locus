package persistence

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestSQLiteCommandRepository_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	cmd := entity.Command{
		Title:     "Test task",
		Status:    entity.StatusNotStarted,
		StageId:   entity.StagePlan,
		SortIndex: 0,
		CreatedAt: time.Now().UTC(),
	}
	created, err := repo.Create(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("ID should be assigned")
	}
	if created.Title != cmd.Title {
		t.Fatalf("Title = %q, want %q", created.Title, cmd.Title)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != cmd.Title || got.StageId != cmd.StageId {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestSQLiteCommandRepository_Create_ZeroCreatedAt(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	cmd := entity.Command{
		Title:   "No time",
		Status:  entity.StatusNotStarted,
		StageId: entity.StagePlan,
		// CreatedAt zero;repo should default it.
	}
	created, err := repo.Create(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set by repo when zero")
	}
}

func TestSQLiteCommandRepository_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	_, err := repo.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want wrapping sql.ErrNoRows", err)
	}
}

func TestSQLiteCommandRepository_List_All(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	for _, stage := range []entity.StageId{entity.StagePlan, entity.StageExecute} {
		_, err := repo.Create(ctx, entity.Command{
			Title: "T", Status: entity.StatusNotStarted, StageId: stage, CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	all, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("len = %d, want 2", len(all))
	}
}

func TestSQLiteCommandRepository_List_Filtered(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, entity.Command{Title: "A", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC()})
	_, _ = repo.Create(ctx, entity.Command{Title: "B", Status: entity.StatusNotStarted, StageId: entity.StageExecute, CreatedAt: time.Now().UTC()})

	sid := entity.StagePlan
	result, err := repo.List(ctx, &sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Title != "A" {
		t.Fatalf("unexpected: %v", result)
	}
}

func TestSQLiteCommandRepository_Update(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, entity.Command{
		Title: "Old", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC(),
	})
	created.Title = "New"
	created.Status = entity.StatusInProgress
	if err := repo.Update(ctx, created); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, created.ID)
	if got.Title != "New" || got.Status != entity.StatusInProgress {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestSQLiteCommandRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, entity.Command{
		Title: "T", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC(),
	})
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	_, err := repo.Get(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestSQLiteCommandRepository_Source_Default(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, entity.Command{
		Title: "Manual", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, created.ID)
	if got.Source != entity.SourceManual {
		t.Errorf("Source = %q, want %q", got.Source, entity.SourceManual)
	}
}

func TestSQLiteCommandRepository_Claude_Source(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, entity.Command{
		Title: "Dynamic", Status: entity.StatusNotStarted, StageId: entity.StageExecute,
		Source: entity.SourceClaude, SessionID: "sess-abc", CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, created.ID)
	if got.Source != entity.SourceClaude {
		t.Errorf("Source = %q, want %q", got.Source, entity.SourceClaude)
	}
	if got.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc", got.SessionID)
	}
	if got.ArchivedAt != nil {
		t.Error("ArchivedAt should be nil before archival")
	}
}

func TestSQLiteCommandRepository_ArchiveSession(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	// Create two dynamic items for session s1 and one manual item.
	_, _ = repo.Create(ctx, entity.Command{
		Title: "D1", Status: entity.StatusNotStarted, StageId: entity.StageExecute,
		Source: entity.SourceClaude, SessionID: "s1", CreatedAt: time.Now().UTC(),
	})
	_, _ = repo.Create(ctx, entity.Command{
		Title: "D2", Status: entity.StatusNotStarted, StageId: entity.StageCheck,
		Source: entity.SourceClaude, SessionID: "s1", CreatedAt: time.Now().UTC(),
	})
	_, _ = repo.Create(ctx, entity.Command{
		Title: "Manual", Status: entity.StatusNotStarted, StageId: entity.StagePlan,
		Source: entity.SourceManual, CreatedAt: time.Now().UTC(),
	})

	if err := repo.ArchiveSession(ctx, "s1", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	all, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Only the manual item should remain visible.
	if len(all) != 1 {
		t.Fatalf("List after archive: want 1, got %d", len(all))
	}
	if all[0].Title != "Manual" {
		t.Errorf("unexpected item after archive: %q", all[0].Title)
	}
}

func TestSQLiteCommandRepository_Reorder(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()

	c1, _ := repo.Create(ctx, entity.Command{Title: "A", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC()})
	c2, _ := repo.Create(ctx, entity.Command{Title: "B", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC()})

	// Reverse order.
	err := repo.Reorder(ctx, map[entity.StageId][]int64{
		entity.StagePlan: {c2.ID, c1.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	all, _ := repo.List(ctx, nil)
	// After reorder, c2 should have sort_index=0 and come first.
	if all[0].ID != c2.ID {
		t.Fatalf("reorder not applied: got ID=%d first, want %d", all[0].ID, c2.ID)
	}
}
