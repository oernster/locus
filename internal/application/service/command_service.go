package service

import (
	"context"
	"fmt"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// CommandService handles command (task) business logic.
type CommandService struct {
	repo repository.CommandRepository
}

// NewCommandService creates a CommandService backed by the given repository.
func NewCommandService(repo repository.CommandRepository) *CommandService {
	return &CommandService{repo: repo}
}

// List returns all commands, optionally filtered to a single stage.
func (s *CommandService) List(ctx context.Context, stageId string) ([]dto.CommandDTO, error) {
	var filter *entity.StageId
	if stageId != "" {
		sid := entity.StageId(stageId)
		filter = &sid
	}
	cmds, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CommandDTO, len(cmds))
	for i, c := range cmds {
		out[i] = toCommandDTO(c)
	}
	return out, nil
}

// Get returns a single command by ID.
func (s *CommandService) Get(ctx context.Context, id int64) (dto.CommandDTO, error) {
	cmd, err := s.repo.Get(ctx, id)
	if err != nil {
		return dto.CommandDTO{}, err
	}
	return toCommandDTO(cmd), nil
}

// Create adds a new command on the board.
func (s *CommandService) Create(ctx context.Context, title, stageId string) (dto.CommandDTO, error) {
	if title == "" {
		return dto.CommandDTO{}, fmt.Errorf("title must not be empty")
	}
	sid := entity.StageId(stageId)
	if !validStage(sid) {
		return dto.CommandDTO{}, fmt.Errorf("invalid stage_id: %q", stageId)
	}
	cmd := entity.Command{
		Title:   title,
		Status:  entity.StatusNotStarted,
		StageId: sid,
	}
	created, err := s.repo.Create(ctx, cmd)
	if err != nil {
		return dto.CommandDTO{}, err
	}
	return toCommandDTO(created), nil
}

// Update modifies an existing command.
func (s *CommandService) Update(ctx context.Context, id int64, title, status, stageId string) (dto.CommandDTO, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return dto.CommandDTO{}, err
	}

	if title != "" {
		existing.Title = title
	}
	if status != "" {
		existing.Status = entity.Status(status)
	}
	if stageId != "" {
		sid := entity.StageId(stageId)
		if !validStage(sid) {
			return dto.CommandDTO{}, fmt.Errorf("invalid stage_id: %q", stageId)
		}
		existing.StageId = sid
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return dto.CommandDTO{}, err
	}
	return toCommandDTO(existing), nil
}

// Delete removes a command by ID.
func (s *CommandService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Reorder persists a new sort order for commands across stages.
func (s *CommandService) Reorder(ctx context.Context, byStageId map[string][]int64) error {
	mapped := make(map[entity.StageId][]int64, len(byStageId))
	for k, v := range byStageId {
		mapped[entity.StageId(k)] = v
	}
	return s.repo.Reorder(ctx, mapped)
}

func toCommandDTO(c entity.Command) dto.CommandDTO {
	source := c.Source
	if source == "" {
		source = entity.SourceManual
	}
	return dto.CommandDTO{
		ID:        c.ID,
		Title:     c.Title,
		Status:    string(c.Status),
		StageId:   string(c.StageId),
		SortIndex: c.SortIndex,
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Source:    source,
		SessionID: c.SessionID,
	}
}

func validStage(s entity.StageId) bool {
	for _, v := range entity.Stages {
		if v == s {
			return true
		}
	}
	return false
}
