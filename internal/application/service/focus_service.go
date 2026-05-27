package service

import (
	"context"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// FocusReader is the interface for querying the focus-reader database.
type FocusReader interface {
	// GetFocusDataForSessions returns aggregated focus data for the supplied
	// locus session windows. Returns dto.FocusDataDTO{Available: false} when
	// the focus-reader database is absent or unreadable.
	GetFocusDataForSessions(sessions []FocusSessionWindow) dto.FocusDataDTO
}

// FocusSessionWindow is a time range derived from a locus session.
type FocusSessionWindow struct {
	StageId   entity.StageId
	StartedAt int64 // Unix seconds
	EndedAt   int64 // Unix seconds
}

// FocusService provides focus insight data by correlating locus sessions with
// the focus-reader SQLite database.
type FocusService struct {
	sessionRepo repository.SessionRepository
	reader      FocusReader
}

// NewFocusService creates a FocusService.
func NewFocusService(sessionRepo repository.SessionRepository, reader FocusReader) *FocusService {
	return &FocusService{sessionRepo: sessionRepo, reader: reader}
}

// GetFocusDataForStage returns focus-reader data correlated with locus sessions
// for the given stage.
func (s *FocusService) GetFocusDataForStage(ctx context.Context, stageId string) (dto.FocusDataDTO, error) {
	sid := entity.StageId(stageId)

	windows, err := gatherWindows(ctx, s.sessionRepo, sid)
	if err != nil {
		return dto.FocusDataDTO{Available: false, StageId: stageId}, nil
	}

	if len(windows) == 0 {
		return dto.FocusDataDTO{Available: true, StageId: stageId, Apps: []dto.AppFocusDTO{}}, nil
	}

	result := s.reader.GetFocusDataForSessions(windows)
	result.StageId = stageId

	return result, nil
}

// rollingWindowHours is how far back to look for focus data when no locus
// sessions exist for a stage.
const rollingWindowHours = 2

// gatherWindows collects locus session windows for a stage. If no sessions
// exist for the stage it returns a single rolling window covering the last
// rollingWindowHours hours so real-time focus data is visible immediately.
func gatherWindows(ctx context.Context, repo repository.SessionRepository, sid entity.StageId) ([]FocusSessionWindow, error) {
	// Pull all sessions from the beginning of time.
	allSessions, err := repo.ListByTimeRange(ctx, time.Unix(0, 0).UTC(), time.Now().UTC())
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var windows []FocusSessionWindow
	for _, sess := range allSessions {
		if sess.StageId != sid {
			continue
		}
		endedAt := now.Unix()
		if sess.EndedAt != nil {
			endedAt = sess.EndedAt.Unix()
		}
		windows = append(windows, FocusSessionWindow{
			StageId:   sess.StageId,
			StartedAt: sess.StartedAt.Unix(),
			EndedAt:   endedAt,
		})
	}

	// Fallback: no sessions yet for this stage -- show a rolling recent window
	// so the focus tracker output is visible immediately.
	if len(windows) == 0 {
		windows = []FocusSessionWindow{{
			StageId:   sid,
			StartedAt: now.Add(-rollingWindowHours * time.Hour).Unix(),
			EndedAt:   now.Unix(),
		}}
	}

	return windows, nil
}
