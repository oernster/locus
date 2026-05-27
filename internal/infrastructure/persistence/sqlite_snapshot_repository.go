package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// SQLiteSnapshotRepository implements repository.SnapshotRepository.
type SQLiteSnapshotRepository struct {
	db *sql.DB
}

// NewSQLiteSnapshotRepository creates the repository.
func NewSQLiteSnapshotRepository(db *sql.DB) *SQLiteSnapshotRepository {
	return &SQLiteSnapshotRepository{db: db}
}

// List returns all snapshots ordered by saved_at DESC.
func (r *SQLiteSnapshotRepository) List(ctx context.Context) ([]entity.Snapshot, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, data, hash, saved_at FROM snapshots ORDER BY saved_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSnapshots(rows)
}

// Get returns a single snapshot by ID.
func (r *SQLiteSnapshotRepository) Get(ctx context.Context, id int64) (entity.Snapshot, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, data, hash, saved_at FROM snapshots WHERE id=?`, id)
	return scanSnapshot(row)
}

// Create inserts a new snapshot.
func (r *SQLiteSnapshotRepository) Create(ctx context.Context, s entity.Snapshot) (entity.Snapshot, error) {
	if s.SavedAt.IsZero() {
		s.SavedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO snapshots (name, data, hash, saved_at) VALUES (?, ?, ?, ?)`,
		s.Name, s.Data, s.Hash, s.SavedAt.Unix())
	if err != nil {
		return entity.Snapshot{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return entity.Snapshot{}, err
	}
	s.ID = id
	return s, nil
}

// Update persists changes to an existing snapshot (typically a rename).
func (r *SQLiteSnapshotRepository) Update(ctx context.Context, s entity.Snapshot) (entity.Snapshot, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE snapshots SET name=?, data=?, hash=?, saved_at=? WHERE id=?`,
		s.Name, s.Data, s.Hash, s.SavedAt.Unix(), s.ID)
	if err != nil {
		return entity.Snapshot{}, err
	}
	return s, nil
}

// Delete removes a snapshot.
func (r *SQLiteSnapshotRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM snapshots WHERE id=?`, id)
	return err
}

// FindByHash returns the snapshot with the given content hash, or nil.
func (r *SQLiteSnapshotRepository) FindByHash(ctx context.Context, hash string) (*entity.Snapshot, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, data, hash, saved_at FROM snapshots WHERE hash=? LIMIT 1`, hash)
	s, err := scanSnapshot(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type snapshotScanner interface {
	Scan(dest ...any) error
}

func scanSnapshot(row snapshotScanner) (entity.Snapshot, error) {
	var id, savedAt int64
	var name, data, hash string
	if err := row.Scan(&id, &name, &data, &hash, &savedAt); err != nil {
		return entity.Snapshot{}, err
	}
	return entity.Snapshot{
		ID:      id,
		Name:    name,
		Data:    data,
		Hash:    hash,
		SavedAt: time.Unix(savedAt, 0).UTC(),
	}, nil
}

func scanSnapshots(rows *sql.Rows) ([]entity.Snapshot, error) {
	var snaps []entity.Snapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snaps, nil
}

