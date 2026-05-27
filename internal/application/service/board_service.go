package service

import (
	"context"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// BoardService handles board-level business logic.
type BoardService struct {
	boardRepo   repository.BoardRepository
	commandRepo repository.CommandRepository
}

// NewBoardService creates a BoardService.
func NewBoardService(boardRepo repository.BoardRepository, commandRepo repository.CommandRepository) *BoardService {
	return &BoardService{boardRepo: boardRepo, commandRepo: commandRepo}
}

// Get returns the current board state, creating the default record if absent.
func (s *BoardService) Get(ctx context.Context) (dto.BoardDTO, error) {
	if !s.boardRepo.Exists(ctx) {
		_, err := s.boardRepo.Update(ctx, entity.BoardState{})
		if err != nil {
			return dto.BoardDTO{}, err
		}
	}
	b, err := s.boardRepo.Get(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	isEmpty, err := s.isEmpty(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	return toBoardDTO(b, isEmpty), nil
}

// UpdateName renames the board.
func (s *BoardService) UpdateName(ctx context.Context, name string) (dto.BoardDTO, error) {
	b, err := s.boardRepo.Get(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	b.Name = name
	b.UserNamed = true
	updated, err := s.boardRepo.Update(ctx, b)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	isEmpty, err := s.isEmpty(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	return toBoardDTO(updated, isEmpty), nil
}

// UpdateStageLabels replaces the stage label overrides.
func (s *BoardService) UpdateStageLabels(ctx context.Context, labels map[string]string) (dto.BoardDTO, error) {
	b, err := s.boardRepo.Get(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	b.StageLabels = labels
	updated, err := s.boardRepo.Update(ctx, b)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	isEmpty, err := s.isEmpty(ctx)
	if err != nil {
		return dto.BoardDTO{}, err
	}
	return toBoardDTO(updated, isEmpty), nil
}

// Reset clears all commands and resets the board to an empty unnamed state.
func (s *BoardService) Reset(ctx context.Context) error {
	cmds, err := s.commandRepo.List(ctx, nil)
	if err != nil {
		return err
	}
	for _, c := range cmds {
		if err := s.commandRepo.Delete(ctx, c.ID); err != nil {
			return err
		}
	}
	empty := entity.BoardState{Name: "", UserNamed: false, StageLabels: nil}
	_, err = s.boardRepo.Update(ctx, empty)
	return err
}

// IsEmpty reports whether the board has no commands.
func (s *BoardService) IsEmpty(ctx context.Context) (bool, error) {
	return s.isEmpty(ctx)
}

func (s *BoardService) isEmpty(ctx context.Context) (bool, error) {
	cmds, err := s.commandRepo.List(ctx, nil)
	if err != nil {
		return false, err
	}
	return len(cmds) == 0, nil
}

func toBoardDTO(b entity.BoardState, isEmpty bool) dto.BoardDTO {
	return dto.BoardDTO{
		Name:         b.Name,
		UserNamed:    b.UserNamed,
		IsNewUnnamed: !b.UserNamed && b.Name == "",
		IsEmpty:      isEmpty,
		StageLabels:  b.StageLabels,
	}
}
