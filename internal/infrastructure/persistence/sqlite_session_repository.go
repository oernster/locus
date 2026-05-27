package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// SQLiteSessionRepository implements repository.SessionRepository.
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSQLiteSessionRepository creates the repository.
func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

// GetActive returns the single active session (ended_at IS NULL), or nil.
func (r *SQLiteSessionRepository) GetActive(ctx context.Context) (*entity.Session, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, command_id, stage_id, started_at, ended_at
		 FROM sessions WHERE ended_at IS NULL LIMIT 1`)
	s, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetLatestByStageId returns the most recent session per stage (any ended_at).
func (r *SQLiteSessionRepository) GetLatestByStageId(ctx context.Context) (map[entity.StageId]*entity.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, command_id, stage_id, started_at, ended_at
		 FROM sessions
		 WHERE id IN (
		   SELECT MAX(id) FROM sessions GROUP BY stage_id
		 )`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[entity.StageId]*entity.Session)
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		copy := s
		result[s.StageId] = &copy
	}
	return result, rows.Err()
}

// Create inserts a new session.
func (r *SQLiteSessionRepository) Create(ctx context.Context, s entity.Session) (entity.Session, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (command_id, stage_id, started_at, ended_at) VALUES (?, ?, ?, ?)`,
		s.CommandID, string(s.StageId), s.StartedAt.Unix(), encodeNullTime(s.EndedAt))
	if err != nil {
		return entity.Session{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Session{}, err
	}
	s.ID = id
	return s, nil
}

// Update persists changes to an existing session.
func (r *SQLiteSessionRepository) Update(ctx context.Context, s entity.Session) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET command_id=?, stage_id=?, started_at=?, ended_at=? WHERE id=?`,
		s.CommandID, string(s.StageId), s.StartedAt.Unix(), encodeNullTime(s.EndedAt), s.ID)
	return err
}

// ListByTimeRange returns sessions with started_at in [from, to].
func (r *SQLiteSessionRepository) ListByTimeRange(ctx context.Context, from, to time.Time) ([]entity.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, command_id, stage_id, started_at, ended_at
		 FROM sessions WHERE started_at >= ? AND started_at <= ? ORDER BY started_at`,
		from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []entity.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSession(row sessionScanner) (entity.Session, error) {
	var (
		id, commandID, startedAt int64
		stageId                  string
		endedAt                  sql.NullInt64
	)
	if err := row.Scan(&id, &commandID, &stageId, &startedAt, &endedAt); err != nil {
		return entity.Session{}, err
	}
	s := entity.Session{
		ID:        id,
		CommandID: commandID,
		StageId:   entity.StageId(stageId),
		StartedAt: time.Unix(startedAt, 0).UTC(),
	}
	if endedAt.Valid {
		t := time.Unix(endedAt.Int64, 0).UTC()
		s.EndedAt = &t
	}
	return s, nil
}

func encodeNullTime(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}
