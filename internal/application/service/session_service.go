package service

import (
	"context"
	"fmt"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// SessionService handles session tracking business logic.
type SessionService struct {
	sessionRepo repository.SessionRepository
	commandRepo repository.CommandRepository
}

// NewSessionService creates a SessionService.
func NewSessionService(sessionRepo repository.SessionRepository, commandRepo repository.CommandRepository) *SessionService {
	return &SessionService{sessionRepo: sessionRepo, commandRepo: commandRepo}
}

// GetActive returns the currently active session, or an inactive DTO if none.
func (s *SessionService) GetActive(ctx context.Context) (dto.SessionDTO, error) {
	sess, err := s.sessionRepo.GetActive(ctx)
	if err != nil {
		return dto.SessionDTO{}, err
	}
	if sess == nil {
		return dto.SessionDTO{Active: false}, nil
	}
	return toSessionDTO(*sess), nil
}

// GetLatestByStageId returns the most-recent session per stage.
func (s *SessionService) GetLatestByStageId(ctx context.Context) (map[string]dto.SessionDTO, error) {
	latest, err := s.sessionRepo.GetLatestByStageId(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]dto.SessionDTO, len(latest))
	for stageId, sess := range latest {
		if sess != nil {
			out[string(stageId)] = toSessionDTO(*sess)
		}
	}
	return out, nil
}

// Start creates a new session for the given command. Any existing active session
// is stopped first.
func (s *SessionService) Start(ctx context.Context, commandID int64) (dto.SessionDTO, error) {
	// Resolve the command to find its current stage.
	cmd, err := s.commandRepo.Get(ctx, commandID)
	if err != nil {
		return dto.SessionDTO{}, fmt.Errorf("command not found: %w", err)
	}

	// Stop any active session before starting a new one.
	active, err := s.sessionRepo.GetActive(ctx)
	if err != nil {
		return dto.SessionDTO{}, err
	}
	if active != nil {
		now := time.Now().UTC()
		active.EndedAt = &now
		if err := s.sessionRepo.Update(ctx, *active); err != nil {
			return dto.SessionDTO{}, err
		}
	}

	sess := entity.Session{
		CommandID: commandID,
		StageId:   cmd.StageId,
		StartedAt: time.Now().UTC(),
	}
	created, err := s.sessionRepo.Create(ctx, sess)
	if err != nil {
		return dto.SessionDTO{}, err
	}
	return toSessionDTO(created), nil
}

// Stop ends the currently active session.
func (s *SessionService) Stop(ctx context.Context) error {
	active, err := s.sessionRepo.GetActive(ctx)
	if err != nil {
		return err
	}
	if active == nil {
		return nil
	}
	now := time.Now().UTC()
	active.EndedAt = &now
	return s.sessionRepo.Update(ctx, *active)
}

func toSessionDTO(s entity.Session) dto.SessionDTO {
	d := dto.SessionDTO{
		Active:    s.EndedAt == nil,
		ID:        s.ID,
		CommandID: s.CommandID,
		StageId:   string(s.StageId),
		StartedAt: s.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if s.EndedAt != nil {
		d.EndedAt = s.EndedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return d
}
