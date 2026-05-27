package repository

import (
	"context"

	"github.com/oernster/locus/internal/domain/entity"
)

// BoardRepository manages the singleton BoardState record.
type BoardRepository interface {
	Get(ctx context.Context) (entity.BoardState, error)
	Update(ctx context.Context, b entity.BoardState) (entity.BoardState, error)
	Exists(ctx context.Context) bool
}
