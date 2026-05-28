package repository

import (
	"context"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

// CommandRepository defines persistence operations for Commands.
type CommandRepository interface {
	List(ctx context.Context, stageId *entity.StageId) ([]entity.Command, error)
	Get(ctx context.Context, id int64) (entity.Command, error)
	Create(ctx context.Context, cmd entity.Command) (entity.Command, error)
	Update(ctx context.Context, cmd entity.Command) error
	Delete(ctx context.Context, id int64) error
	Reorder(ctx context.Context, byStageId map[entity.StageId][]int64) error
	// ArchiveSession soft-deletes all Claude-sourced commands belonging to sessionID
	// by setting their archived_at timestamp. Archived commands are excluded from List.
	ArchiveSession(ctx context.Context, sessionID string, archivedAt time.Time) error
}
