package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func seedCommand(t *testing.T, repo *SQLiteCommandRepository, stage entity.StageId) entity.Command {
	t.Helper()
	c, err := repo.Create(context.Background(), entity.Command{
		Title: "T", Status: entity.StatusNotStarted, StageId: stage, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed command: %v", err)
	}
	return c
}

func TestSQLiteSessionRepository_GetActive_None(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSessionRepository(db)
	s, err := repo.GetActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatal("expected nil active session")
	}
}

func TestSQLiteSessionRepository_CreateAndGetActive(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteSessionRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	sess := entity.Session{
		CommandID: cmd.ID,
		StageId:   entity.StagePlan,
		StartedAt: time.Now().UTC(),
	}
	created, err := repo.Create(ctx, sess)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("ID not assigned")
	}

	active, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != created.ID {
		t.Fatalf("unexpected active: %v", active)
	}
}

func TestSQLiteSessionRepository_Update_EndSession(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteSessionRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StageExecute)
	created, _ := repo.Create(ctx, entity.Session{
		CommandID: cmd.ID, StageId: entity.StageExecute, StartedAt: time.Now().UTC(),
	})

	now := time.Now().UTC()
	created.EndedAt = &now
	if err := repo.Update(ctx, created); err != nil {
		t.Fatal(err)
	}

	// No active sessions after ending.
	active, _ := repo.GetActive(ctx)
	if active != nil {
		t.Fatal("expected no active session after update")
	}
}

func TestSQLiteSessionRepository_GetLatestByStageId(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteSessionRepository(db)
	ctx := context.Background()

	cmd1 := seedCommand(t, cmdRepo, entity.StagePlan)
	cmd2 := seedCommand(t, cmdRepo, entity.StageCheck)

	now := time.Now().UTC()
	ended := now.Add(time.Hour)

	_, _ = repo.Create(ctx, entity.Session{CommandID: cmd1.ID, StageId: entity.StagePlan, StartedAt: now, EndedAt: &ended})
	_, _ = repo.Create(ctx, entity.Session{CommandID: cmd2.ID, StageId: entity.StageCheck, StartedAt: now})

	latest, err := repo.GetLatestByStageId(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := latest[entity.StagePlan]; !ok {
		t.Fatal("PLAN missing from latest")
	}
	if _, ok := latest[entity.StageCheck]; !ok {
		t.Fatal("CHECK missing from latest")
	}
}

func TestSQLiteSessionRepository_ListByTimeRange(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteSessionRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	now := time.Now().UTC()
	_, _ = repo.Create(ctx, entity.Session{CommandID: cmd.ID, StageId: entity.StagePlan, StartedAt: now})

	from := now.Add(-time.Minute)
	to := now.Add(time.Minute)
	sessions, err := repo.ListByTimeRange(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d, want 1", len(sessions))
	}
}

func TestSQLiteSessionRepository_ListByTimeRange_OutOfRange(t *testing.T) {
	db := newTestDB(t)
	cmdRepo := NewSQLiteCommandRepository(db)
	repo := NewSQLiteSessionRepository(db)
	ctx := context.Background()

	cmd := seedCommand(t, cmdRepo, entity.StagePlan)
	now := time.Now().UTC()
	_, _ = repo.Create(ctx, entity.Session{CommandID: cmd.ID, StageId: entity.StagePlan, StartedAt: now})

	// Range before the session.
	from := now.Add(-2 * time.Hour)
	to := now.Add(-time.Hour)
	sessions, err := repo.ListByTimeRange(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestEncodeNullTime_Nil(t *testing.T) {
	n := encodeNullTime(nil)
	if n.Valid {
		t.Fatal("nil time should produce NullInt64{Valid:false}")
	}
}

func TestEncodeNullTime_NonNil(t *testing.T) {
	ts := time.Now()
	n := encodeNullTime(&ts)
	if !n.Valid {
		t.Fatal("non-nil time should produce valid NullInt64")
	}
	if n.Int64 != ts.Unix() {
		t.Fatalf("Int64 = %d, want %d", n.Int64, ts.Unix())
	}
}
