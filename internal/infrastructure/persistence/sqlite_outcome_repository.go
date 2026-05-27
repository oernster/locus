package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// SQLiteOutcomeRepository implements repository.OutcomeRepository.
type SQLiteOutcomeRepository struct {
	db *sql.DB
}

// NewSQLiteOutcomeRepository creates the repository.
func NewSQLiteOutcomeRepository(db *sql.DB) *SQLiteOutcomeRepository {
	return &SQLiteOutcomeRepository{db: db}
}

// ListByCommandID returns all outcomes for a command, newest first.
func (r *SQLiteOutcomeRepository) ListByCommandID(ctx context.Context, commandID int64) ([]entity.Outcome, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, command_id, note, created_at FROM outcomes WHERE command_id=? ORDER BY created_at DESC`,
		commandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []entity.Outcome
	for rows.Next() {
		var id, cid, createdAt int64
		var note string
		if err := rows.Scan(&id, &cid, &note, &createdAt); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, entity.Outcome{
			ID:        id,
			CommandID: cid,
			Note:      note,
			CreatedAt: time.Unix(createdAt, 0).UTC(),
		})
	}
	return outcomes, rows.Err()
}

// Create inserts a new outcome.
func (r *SQLiteOutcomeRepository) Create(ctx context.Context, o entity.Outcome) (entity.Outcome, error) {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO outcomes (command_id, note, created_at) VALUES (?, ?, ?)`,
		o.CommandID, o.Note, o.CreatedAt.Unix())
	if err != nil {
		return entity.Outcome{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Outcome{}, err
	}
	o.ID = id
	return o, nil
}

// Delete removes an outcome by ID.
func (r *SQLiteOutcomeRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM outcomes WHERE id=?`, id)
	return err
}

