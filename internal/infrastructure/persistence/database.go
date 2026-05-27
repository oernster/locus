package persistence

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS commands (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT    NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'Not Started',
    stage_id   TEXT    NOT NULL DEFAULT 'PLAN',
    sort_index INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    command_id INTEGER NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
    stage_id   TEXT    NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at   INTEGER
);
CREATE TABLE IF NOT EXISTS outcomes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    command_id INTEGER NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
    note       TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS board_state (
    id                INTEGER PRIMARY KEY DEFAULT 1,
    name              TEXT NOT NULL DEFAULT '',
    user_named        INTEGER NOT NULL DEFAULT 0,
    stage_labels_json TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS snapshots (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL,
    data     TEXT    NOT NULL,
    hash     TEXT    NOT NULL,
    saved_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS focus_sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    exe_path   TEXT    NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at   INTEGER
);
CREATE INDEX IF NOT EXISTS idx_sessions_command ON sessions(command_id);
CREATE INDEX IF NOT EXISTS idx_outcomes_command ON outcomes(command_id);
CREATE INDEX IF NOT EXISTS idx_focus_sessions_time ON focus_sessions(started_at);
PRAGMA foreign_keys = ON;
`

// Open opens or creates the SQLite database at dbPath and applies the schema.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
