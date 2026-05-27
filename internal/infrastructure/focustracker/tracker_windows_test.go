//go:build windows

package focustracker

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SQLite DB with the focus_sessions table.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS focus_sessions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			exe_path   TEXT    NOT NULL,
			started_at INTEGER NOT NULL,
			ended_at   INTEGER
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func countOpen(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM focus_sessions WHERE ended_at IS NULL`).Scan(&n)
	return n
}

func TestNew_DefaultForegroundFn(t *testing.T) {
	db := newTestDB(t)
	tr := New(db)
	if tr.db != db {
		t.Fatal("db not set")
	}
	if tr.foregroundExeFn == nil {
		t.Fatal("foregroundExeFn should be set to foregroundExe")
	}
	if tr.stop == nil {
		t.Fatal("stop channel should be initialised")
	}
}

func TestTracker_CloseStale(t *testing.T) {
	db := newTestDB(t)
	// Insert a stale open session.
	_, err := db.Exec(`INSERT INTO focus_sessions (exe_path, started_at, ended_at) VALUES ('x.exe', ?, NULL)`,
		time.Now().Unix()-100)
	if err != nil {
		t.Fatal(err)
	}
	if countOpen(t, db) != 1 {
		t.Fatal("expected 1 open session before closeStale")
	}

	tr := New(db)
	tr.closeStale()

	if countOpen(t, db) != 0 {
		t.Fatal("expected 0 open sessions after closeStale")
	}
}

func TestTracker_StartAndEndSession(t *testing.T) {
	db := newTestDB(t)
	tr := New(db)

	id := tr.startSession(`C:\code.exe`)
	if id == 0 {
		t.Fatal("startSession should return non-zero ID")
	}
	if countOpen(t, db) != 1 {
		t.Fatal("expected 1 open session after start")
	}

	tr.endSession(id)
	if countOpen(t, db) != 0 {
		t.Fatal("expected 0 open sessions after endSession")
	}
}

func TestTracker_Run_SwitchesSession(t *testing.T) {
	db := newTestDB(t)

	calls := make(chan string, 10)
	// Simulate: exe1, exe1 (same, skip), exe2, then stop.
	calls <- `C:\app1.exe`
	calls <- `C:\app1.exe`
	calls <- `C:\app2.exe`

	tr := &Tracker{
		db:   db,
		stop: make(chan struct{}),
		foregroundExeFn: func() string {
			select {
			case exe := <-calls:
				return exe
			default:
				return ""
			}
		},
	}

	go tr.run()
	time.Sleep(2 * pollInterval)
	close(tr.stop)
	time.Sleep(pollInterval)

	// At least one session should have been recorded.
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM focus_sessions`).Scan(&count)
	if count == 0 {
		t.Fatal("expected at least 1 focus session recorded")
	}
}

func TestTracker_Stop_EndsCurrentSession(t *testing.T) {
	db := newTestDB(t)

	callCount := 0
	tr := &Tracker{
		db:   db,
		stop: make(chan struct{}),
		foregroundExeFn: func() string {
			callCount++
			if callCount == 1 {
				return `C:\editor.exe`
			}
			return ""
		},
	}

	go tr.run()
	time.Sleep(2 * pollInterval)

	// Stop the tracker.
	tr.Stop()
	time.Sleep(pollInterval)

	// After stop, the active session should be ended.
	if countOpen(t, db) != 0 {
		t.Fatalf("expected 0 open sessions after Stop, got %d", countOpen(t, db))
	}
}

func TestTracker_Start_ClosesStaleAndRuns(t *testing.T) {
	db := newTestDB(t)
	// Plant a stale session.
	_, _ = db.Exec(`INSERT INTO focus_sessions (exe_path, started_at) VALUES ('old.exe', ?)`, time.Now().Unix()-3600)

	tr := &Tracker{
		db:              db,
		stop:            make(chan struct{}),
		foregroundExeFn: func() string { return "" },
	}
	tr.Start()
	time.Sleep(pollInterval)
	close(tr.stop)

	if countOpen(t, db) != 0 {
		t.Fatal("stale session should be closed by Start")
	}
}
