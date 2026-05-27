package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/oernster/locus/internal/domain/entity"
)

// SQLiteBoardRepository implements repository.BoardRepository.
type SQLiteBoardRepository struct {
	db *sql.DB
}

// NewSQLiteBoardRepository creates the repository.
func NewSQLiteBoardRepository(db *sql.DB) *SQLiteBoardRepository {
	return &SQLiteBoardRepository{db: db}
}

// Exists reports whether the board_state row (id=1) is present.
func (r *SQLiteBoardRepository) Exists(ctx context.Context) bool {
	var count int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_state WHERE id=1`).Scan(&count)
	return count > 0
}

// Get returns the board state, inserting a default row if absent.
func (r *SQLiteBoardRepository) Get(ctx context.Context) (entity.BoardState, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT name, user_named, stage_labels_json FROM board_state WHERE id=1`)
	var name, labelsJSON string
	var userNamed int
	if err := row.Scan(&name, &userNamed, &labelsJSON); err != nil {
		if err == sql.ErrNoRows {
			return entity.BoardState{}, nil
		}
		return entity.BoardState{}, err
	}
	labels, err := decodeLabels(labelsJSON)
	if err != nil {
		return entity.BoardState{}, err
	}
	return entity.BoardState{
		Name:        name,
		UserNamed:   userNamed == 1,
		StageLabels: labels,
	}, nil
}

// Update upserts the board state row.
func (r *SQLiteBoardRepository) Update(ctx context.Context, b entity.BoardState) (entity.BoardState, error) {
	labelsJSON, err := encodeLabels(b.StageLabels)
	if err != nil {
		return entity.BoardState{}, err
	}
	userNamed := 0
	if b.UserNamed {
		userNamed = 1
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO board_state (id, name, user_named, stage_labels_json) VALUES (1, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, user_named=excluded.user_named, stage_labels_json=excluded.stage_labels_json`,
		b.Name, userNamed, labelsJSON)
	if err != nil {
		return entity.BoardState{}, err
	}
	return b, nil
}

func encodeLabels(labels map[string]string) (string, error) {
	if labels == nil {
		return "{}", nil
	}
	b, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeLabels(s string) (map[string]string, error) {
	if s == "" || s == "{}" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}
