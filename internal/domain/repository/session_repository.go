package repository

import (
	"context"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// SessionRepository defines persistence operations for Sessions.
type SessionRepository interface {
	GetActive(ctx context.Context) (*entity.Session, error)
	GetLatestByStageId(ctx context.Context) (map[entity.StageId]*entity.Session, error)
	Create(ctx context.Context, s entity.Session) (entity.Session, error)
	Update(ctx context.Context, s entity.Session) error
	ListByTimeRange(ctx context.Context, from, to time.Time) ([]entity.Session, error)
}
