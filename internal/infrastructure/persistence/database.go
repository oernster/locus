package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS commands (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'Not Started',
    stage_id    TEXT    NOT NULL DEFAULT 'PLAN',
    sort_index  INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    source      TEXT    NOT NULL DEFAULT 'manual',
    session_id  TEXT    NOT NULL DEFAULT '',
    archived_at INTEGER
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

// migrations handles existing databases that predate columns added after initial release.
var migrations = []string{
	`ALTER TABLE commands ADD COLUMN source      TEXT    NOT NULL DEFAULT 'manual'`,
	`ALTER TABLE commands ADD COLUMN session_id  TEXT    NOT NULL DEFAULT ''`,
	`ALTER TABLE commands ADD COLUMN archived_at INTEGER`,
}

// Open opens or creates the SQLite database at dbPath, applies the schema and
// any pending column migrations.
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
	if err = runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// runMigrations applies ALTER TABLE statements, silently ignoring
// "duplicate column name" errors that arise when the column already exists.
func runMigrations(db *sql.DB) error {
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("migration %q: %w", m, err)
		}
	}
	return nil
}
