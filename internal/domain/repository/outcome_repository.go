package repository

import (
	"context"

	"github.com/oernster/locus/internal/domain/entity"
)

// OutcomeRepository defines persistence operations for Outcomes.
type OutcomeRepository interface {
	ListByCommandID(ctx context.Context, commandID int64) ([]entity.Outcome, error)
	Create(ctx context.Context, o entity.Outcome) (entity.Outcome, error)
	Delete(ctx context.Context, id int64) error
}
