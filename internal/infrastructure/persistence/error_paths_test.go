package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"

	_ "modernc.org/sqlite"
)

// closedDB returns a DB that has been closed, causing all operations to fail.
func closedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.Close()
	return db
}

// --- CommandRepository error paths ---

func TestSQLiteCommandRepository_List_DBError(t *testing.T) {
	_, err := NewSQLiteCommandRepository(closedDB(t)).List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_List_Filtered_DBError(t *testing.T) {
	sid := entity.StagePlan
	_, err := NewSQLiteCommandRepository(closedDB(t)).List(context.Background(), &sid)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_Create_DBError(t *testing.T) {
	cmd := entity.Command{Title: "T", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC()}
	_, err := NewSQLiteCommandRepository(closedDB(t)).Create(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_Update_DBError(t *testing.T) {
	cmd := entity.Command{ID: 1, Title: "T", Status: entity.StatusNotStarted, StageId: entity.StagePlan}
	if err := NewSQLiteCommandRepository(closedDB(t)).Update(context.Background(), cmd); err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_Delete_DBError(t *testing.T) {
	if err := NewSQLiteCommandRepository(closedDB(t)).Delete(context.Background(), 1); err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_Reorder_DBError(t *testing.T) {
	err := NewSQLiteCommandRepository(closedDB(t)).Reorder(context.Background(), map[entity.StageId][]int64{
		entity.StagePlan: {1},
	})
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteCommandRepository_ArchiveSession_DBError(t *testing.T) {
	err := NewSQLiteCommandRepository(closedDB(t)).ArchiveSession(
		context.Background(), "session-x", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

// --- BoardRepository error paths ---

func TestSQLiteBoardRepository_Get_DBError(t *testing.T) {
	_, err := NewSQLiteBoardRepository(closedDB(t)).Get(context.Background())
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteBoardRepository_Update_DBError(t *testing.T) {
	_, err := NewSQLiteBoardRepository(closedDB(t)).Update(context.Background(), entity.BoardState{Name: "X"})
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteBoardRepository_Exists_DBError(t *testing.T) {
	// Exists swallows errors and returns false.
	exists := NewSQLiteBoardRepository(closedDB(t)).Exists(context.Background())
	if exists {
		t.Fatal("closed DB should return false for Exists")
	}
}

// --- SessionRepository error paths ---

func TestSQLiteSessionRepository_GetActive_DBError(t *testing.T) {
	_, err := NewSQLiteSessionRepository(closedDB(t)).GetActive(context.Background())
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSessionRepository_GetLatestByStageId_DBError(t *testing.T) {
	_, err := NewSQLiteSessionRepository(closedDB(t)).GetLatestByStageId(context.Background())
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSessionRepository_Create_DBError(t *testing.T) {
	sess := entity.Session{CommandID: 1, StageId: entity.StagePlan, StartedAt: time.Now().UTC()}
	_, err := NewSQLiteSessionRepository(closedDB(t)).Create(context.Background(), sess)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSessionRepository_Update_DBError(t *testing.T) {
	sess := entity.Session{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: time.Now().UTC()}
	if err := NewSQLiteSessionRepository(closedDB(t)).Update(context.Background(), sess); err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSessionRepository_ListByTimeRange_DBError(t *testing.T) {
	now := time.Now().UTC()
	_, err := NewSQLiteSessionRepository(closedDB(t)).ListByTimeRange(context.Background(), now, now)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

// --- OutcomeRepository error paths ---

func TestSQLiteOutcomeRepository_ListByCommandID_DBError(t *testing.T) {
	_, err := NewSQLiteOutcomeRepository(closedDB(t)).ListByCommandID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteOutcomeRepository_Create_DBError(t *testing.T) {
	o := entity.Outcome{CommandID: 1, Note: "n", CreatedAt: time.Now().UTC()}
	_, err := NewSQLiteOutcomeRepository(closedDB(t)).Create(context.Background(), o)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteOutcomeRepository_Delete_DBError(t *testing.T) {
	if err := NewSQLiteOutcomeRepository(closedDB(t)).Delete(context.Background(), 1); err == nil {
		t.Fatal("expected error on closed DB")
	}
}

// --- SnapshotRepository error paths ---

func TestSQLiteSnapshotRepository_List_DBError(t *testing.T) {
	_, err := NewSQLiteSnapshotRepository(closedDB(t)).List(context.Background())
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSnapshotRepository_Get_DBError(t *testing.T) {
	_, err := NewSQLiteSnapshotRepository(closedDB(t)).Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSnapshotRepository_Create_DBError(t *testing.T) {
	s := entity.Snapshot{Name: "x", Data: "{}", Hash: "h", SavedAt: time.Now().UTC()}
	_, err := NewSQLiteSnapshotRepository(closedDB(t)).Create(context.Background(), s)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSnapshotRepository_Update_DBError(t *testing.T) {
	s := entity.Snapshot{ID: 1, Name: "x", Data: "{}", Hash: "h", SavedAt: time.Now().UTC()}
	_, err := NewSQLiteSnapshotRepository(closedDB(t)).Update(context.Background(), s)
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSnapshotRepository_Delete_DBError(t *testing.T) {
	if err := NewSQLiteSnapshotRepository(closedDB(t)).Delete(context.Background(), 1); err == nil {
		t.Fatal("expected error on closed DB")
	}
}

func TestSQLiteSnapshotRepository_FindByHash_DBError(t *testing.T) {
	_, err := NewSQLiteSnapshotRepository(closedDB(t)).FindByHash(context.Background(), "h")
	if err == nil {
		t.Fatal("expected error on closed DB")
	}
}

// --- scanCommand non-ErrNoRows error ---

// mockRowScanner injects a scan error for testing scanCommand branches.
type mockRowScanner struct{ err error }

func (m mockRowScanner) Scan(_ ...any) error { return m.err }

func TestScanCommand_ScanError(t *testing.T) {
	sentinel := errScan
	_, err := scanCommand(mockRowScanner{err: sentinel})
	if err != sentinel {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

// errScan is a non-ErrNoRows sentinel for scanCommand tests.
var errScan = sql.ErrConnDone

// Reorder inner loop error: use a transaction on closed DB.
// Already covered by TestSQLiteCommandRepository_Reorder_DBError (BeginTx fails).
// This test covers the ExecContext error inside the transaction loop.
func TestSQLiteCommandRepository_Reorder_ExecError(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteCommandRepository(db)
	ctx := context.Background()
	created, _ := repo.Create(ctx, entity.Command{
		Title: "T", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC(),
	})
	// Drop the commands table to force ExecContext failure inside the transaction.
	if _, err := db.Exec(`DROP TABLE commands`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := repo.Reorder(ctx, map[entity.StageId][]int64{entity.StagePlan: {created.ID}}); err == nil {
		t.Fatal("expected error after drop")
	}
}
