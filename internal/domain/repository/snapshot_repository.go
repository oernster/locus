package repository

import (
	"context"

	"github.com/oernster/locus/internal/domain/entity"
)

// SnapshotRepository manages Snapshot persistence.
type SnapshotRepository interface {
	List(ctx context.Context) ([]entity.Snapshot, error)
	Get(ctx context.Context, id int64) (entity.Snapshot, error)
	Create(ctx context.Context, s entity.Snapshot) (entity.Snapshot, error)
	Update(ctx context.Context, s entity.Snapshot) (entity.Snapshot, error)
	Delete(ctx context.Context, id int64) error
	FindByHash(ctx context.Context, hash string) (*entity.Snapshot, error)
}
