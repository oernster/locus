package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// SQLiteCommandRepository implements repository.CommandRepository using SQLite.
type SQLiteCommandRepository struct {
	db *sql.DB
}

// NewSQLiteCommandRepository creates the repository.
func NewSQLiteCommandRepository(db *sql.DB) *SQLiteCommandRepository {
	return &SQLiteCommandRepository{db: db}
}

// List returns commands, optionally filtered to a single stage.
func (r *SQLiteCommandRepository) List(ctx context.Context, stageId *entity.StageId) ([]entity.Command, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if stageId != nil {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, title, status, stage_id, sort_index, created_at
			 FROM commands WHERE stage_id = ? ORDER BY sort_index, id`,
			string(*stageId))
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, title, status, stage_id, sort_index, created_at
			 FROM commands ORDER BY sort_index, id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommands(rows)
}

// Get returns a single command by ID.
func (r *SQLiteCommandRepository) Get(ctx context.Context, id int64) (entity.Command, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, status, stage_id, sort_index, created_at FROM commands WHERE id = ?`, id)
	return scanCommand(row)
}

// Create inserts a new command and returns it with an assigned ID.
func (r *SQLiteCommandRepository) Create(ctx context.Context, cmd entity.Command) (entity.Command, error) {
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO commands (title, status, stage_id, sort_index, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		cmd.Title, string(cmd.Status), string(cmd.StageId), cmd.SortIndex, cmd.CreatedAt.Unix())
	if err != nil {
		return entity.Command{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Command{}, err
	}
	cmd.ID = id
	return cmd, nil
}

// Update persists changes to an existing command.
func (r *SQLiteCommandRepository) Update(ctx context.Context, cmd entity.Command) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE commands SET title=?, status=?, stage_id=?, sort_index=? WHERE id=?`,
		cmd.Title, string(cmd.Status), string(cmd.StageId), cmd.SortIndex, cmd.ID)
	return err
}

// Delete removes a command and its cascaded outcomes/sessions.
func (r *SQLiteCommandRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM commands WHERE id=?`, id)
	return err
}

// Reorder updates sort_index for each command within its stage.
func (r *SQLiteCommandRepository) Reorder(ctx context.Context, byStageId map[entity.StageId][]int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for stageId, ids := range byStageId {
		for idx, id := range ids {
			if _, err := tx.ExecContext(ctx,
				`UPDATE commands SET sort_index=?, stage_id=? WHERE id=?`,
				idx, string(stageId), id); err != nil {
				return fmt.Errorf("reorder %d: %w", id, err)
			}
		}
	}
	return tx.Commit()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCommand(row rowScanner) (entity.Command, error) {
	var (
		id, createdAt int64
		title, status, stageId string
		sortIndex     int
	)
	if err := row.Scan(&id, &title, &status, &stageId, &sortIndex, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return entity.Command{}, fmt.Errorf("command not found: %w", err)
		}
		return entity.Command{}, err
	}
	return entity.Command{
		ID:        id,
		Title:     title,
		Status:    entity.Status(status),
		StageId:   entity.StageId(stageId),
		SortIndex: sortIndex,
		CreatedAt: time.Unix(createdAt, 0).UTC(),
	}, nil
}

func scanCommands(rows *sql.Rows) ([]entity.Command, error) {
	var cmds []entity.Command
	for rows.Next() {
		c, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	return cmds, rows.Err()
}
