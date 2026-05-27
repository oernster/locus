//go:build windows

package focusreader

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/application/service"
	"github.com/oernster/locus/internal/infrastructure/wininfo"

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
		CREATE INDEX IF NOT EXISTS idx_focus_sessions_time ON focus_sessions(started_at);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// stubAppInfo returns a predictable AppInfo for testing.
func stubAppInfo(exePath string) wininfo.AppInfo {
	names := map[string]string{
		`C:\code.exe`:     "Code Editor",
		`C:\terminal.exe`: "Terminal",
		`C:\Windows\system32\svchost.exe`: "svchost",
	}
	if name, ok := names[exePath]; ok {
		return wininfo.AppInfo{ExePath: exePath, FriendlyName: name}
	}
	return wininfo.AppInfo{ExePath: exePath, FriendlyName: exePath}
}

// insertFocusSession inserts a row into focus_sessions.
func insertFocusSession(t *testing.T, db *sql.DB, exePath string, started, ended int64) {
	t.Helper()
	var endedVal interface{}
	if ended > 0 {
		endedVal = ended
	}
	_, err := db.Exec(
		`INSERT INTO focus_sessions (exe_path, started_at, ended_at) VALUES (?, ?, ?)`,
		exePath, started, endedVal)
	if err != nil {
		t.Fatalf("insert focus session: %v", err)
	}
}

func newTestReader(db *sql.DB) *SQLiteFocusReader {
	return &SQLiteFocusReader{db: db, appInfoFn: stubAppInfo}
}

func TestGetFocusDataForSessions_Empty(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{})
	if !result.Available {
		t.Fatal("expected Available=true for empty windows")
	}
	if len(result.Apps) != 0 {
		t.Fatalf("expected no apps, got %v", result.Apps)
	}
}

func TestGetFocusDataForSessions_SingleApp(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	start := now - 3600
	end := now

	insertFocusSession(t, db, `C:\code.exe`, start, end)

	win := service.FocusSessionWindow{StartedAt: start, EndedAt: end}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if !result.Available {
		t.Fatal("expected Available=true")
	}
	if len(result.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(result.Apps))
	}
	if result.Apps[0].FriendlyName != "Code Editor" {
		t.Fatalf("FriendlyName = %q, want 'Code Editor'", result.Apps[0].FriendlyName)
	}
	if result.TotalSeconds != 3600 {
		t.Fatalf("TotalSeconds = %d, want 3600", result.TotalSeconds)
	}
}

func TestGetFocusDataForSessions_SystemProcessFiltered(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	start := now - 1000
	end := now

	// System process should be filtered out.
	insertFocusSession(t, db, `C:\Windows\system32\svchost.exe`, start, end)

	win := service.FocusSessionWindow{StartedAt: start, EndedAt: end}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if len(result.Apps) != 0 {
		t.Fatalf("system process should be filtered, got %v", result.Apps)
	}
}

func TestGetFocusDataForSessions_IdleGapDetected(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	// Two sessions with a 10-minute gap between them.
	insertFocusSession(t, db, `C:\code.exe`, now-1200, now-900)
	// Gap of 600 seconds (> idleThresholdSeconds=300).
	insertFocusSession(t, db, `C:\code.exe`, now-300, now)

	win := service.FocusSessionWindow{StartedAt: now - 1200, EndedAt: now}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if result.IdleSeconds <= 0 {
		t.Fatalf("expected idle seconds detected, got %d", result.IdleSeconds)
	}
}

func TestGetFocusDataForSessions_IdleTailGap(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	// Session ends 600s before the window end — tail gap > threshold.
	insertFocusSession(t, db, `C:\code.exe`, now-1000, now-700)

	win := service.FocusSessionWindow{StartedAt: now - 1000, EndedAt: now}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if result.IdleSeconds <= 0 {
		t.Fatalf("expected tail idle seconds, got %d", result.IdleSeconds)
	}
}

func TestGetFocusDataForSessions_Clamping(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	// Focus session extends beyond window boundaries — should be clamped.
	insertFocusSession(t, db, `C:\code.exe`, now-5000, now+5000)

	win := service.FocusSessionWindow{StartedAt: now - 600, EndedAt: now}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if len(result.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(result.Apps))
	}
	// Clamped duration should be window size = 600.
	if result.Apps[0].TotalSeconds != 600 {
		t.Fatalf("TotalSeconds = %d, want 600 (clamped)", result.Apps[0].TotalSeconds)
	}
}

func TestGetFocusDataForSessions_DeepWorkNonNegative(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	// Window with nothing in it — deep work should be 0, not negative.
	now := time.Now().Unix()
	win := service.FocusSessionWindow{StartedAt: now - 100, EndedAt: now}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if result.DeepWorkSeconds < 0 {
		t.Fatalf("DeepWorkSeconds = %d, must not be negative", result.DeepWorkSeconds)
	}
}

func TestGetFocusDataForSessions_MultipleWindows(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	// Two disjoint windows, code.exe active in both.
	insertFocusSession(t, db, `C:\code.exe`, now-2000, now-1800)
	insertFocusSession(t, db, `C:\code.exe`, now-1000, now-800)

	wins := []service.FocusSessionWindow{
		{StartedAt: now - 2000, EndedAt: now - 1800},
		{StartedAt: now - 1000, EndedAt: now - 800},
	}
	result := r.GetFocusDataForSessions(wins)
	if len(result.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(result.Apps))
	}
	// 200 + 200 = 400 seconds total.
	if result.Apps[0].TotalSeconds != 400 {
		t.Fatalf("TotalSeconds = %d, want 400", result.Apps[0].TotalSeconds)
	}
	if result.Apps[0].SessionCount != 2 {
		t.Fatalf("SessionCount = %d, want 2", result.Apps[0].SessionCount)
	}
}

func TestGetFocusDataForSessions_MaxAppsInReport(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	start := now - 11000
	end := now

	// Insert 11 distinct apps — only maxAppsInReport=10 should be returned.
	for i := int64(1); i <= 11; i++ {
		exePath := `C:\app` + string(rune('0'+i)) + `.exe`
		insertFocusSession(t, db, exePath, now-i*1000, now-(i-1)*1000-1)
	}

	win := service.FocusSessionWindow{StartedAt: start, EndedAt: end}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if len(result.Apps) > maxAppsInReport {
		t.Fatalf("len(Apps) = %d, want <= %d", len(result.Apps), maxAppsInReport)
	}
}

func TestGetFocusDataForSessions_NilEndedAt(t *testing.T) {
	db := newTestDB(t)
	r := newTestReader(db)

	now := time.Now().Unix()
	// Insert session with NULL ended_at (active session).
	_, err := db.Exec(`INSERT INTO focus_sessions (exe_path, started_at, ended_at) VALUES (?, ?, NULL)`,
		`C:\code.exe`, now-600)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	win := service.FocusSessionWindow{StartedAt: now - 600, EndedAt: now}
	result := r.GetFocusDataForSessions([]service.FocusSessionWindow{win})
	if !result.Available {
		t.Fatal("expected Available=true")
	}
}

// Verify that dto package import is used (ensures AppFocusDTO fields are correct).
var _ dto.FocusDataDTO
