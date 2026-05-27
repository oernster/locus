package service

import (
	"context"
	"fmt"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// OutcomeService handles outcome business logic.
type OutcomeService struct {
	repo repository.OutcomeRepository
}

// NewOutcomeService creates an OutcomeService.
func NewOutcomeService(repo repository.OutcomeRepository) *OutcomeService {
	return &OutcomeService{repo: repo}
}

// ListByCommand returns all outcomes for a command.
func (s *OutcomeService) ListByCommand(ctx context.Context, commandID int64) ([]dto.OutcomeDTO, error) {
	outcomes, err := s.repo.ListByCommandID(ctx, commandID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.OutcomeDTO, len(outcomes))
	for i, o := range outcomes {
		out[i] = toOutcomeDTO(o)
	}
	return out, nil
}

// Create adds a new outcome to a command.
func (s *OutcomeService) Create(ctx context.Context, commandID int64, note string) (dto.OutcomeDTO, error) {
	if note == "" {
		return dto.OutcomeDTO{}, fmt.Errorf("note must not be empty")
	}
	o := entity.Outcome{
		CommandID: commandID,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	created, err := s.repo.Create(ctx, o)
	if err != nil {
		return dto.OutcomeDTO{}, err
	}
	return toOutcomeDTO(created), nil
}

// Delete removes an outcome by ID.
func (s *OutcomeService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func toOutcomeDTO(o entity.Outcome) dto.OutcomeDTO {
	return dto.OutcomeDTO{
		ID:        o.ID,
		CommandID: o.CommandID,
		Note:      o.Note,
		CreatedAt: o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
