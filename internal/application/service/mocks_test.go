package service

import (
	"context"
	"errors"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
)

// errNotFound is a sentinel error for mock repos.
var errNotFound = errors.New("not found")

// mockCommandRepo is an in-memory CommandRepository for testing.
type mockCommandRepo struct {
	cmds        []entity.Command
	nextID      int64
	listErr     error
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	reorderErr  error
	archiveErr  error
}

func (m *mockCommandRepo) List(_ context.Context, stageId *entity.StageId) ([]entity.Command, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if stageId == nil {
		return m.cmds, nil
	}
	var out []entity.Command
	for _, c := range m.cmds {
		if c.StageId == *stageId {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *mockCommandRepo) Get(_ context.Context, id int64) (entity.Command, error) {
	if m.getErr != nil {
		return entity.Command{}, m.getErr
	}
	for _, c := range m.cmds {
		if c.ID == id {
			return c, nil
		}
	}
	return entity.Command{}, errNotFound
}

func (m *mockCommandRepo) Create(_ context.Context, cmd entity.Command) (entity.Command, error) {
	if m.createErr != nil {
		return entity.Command{}, m.createErr
	}
	m.nextID++
	cmd.ID = m.nextID
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now().UTC()
	}
	m.cmds = append(m.cmds, cmd)
	return cmd, nil
}

func (m *mockCommandRepo) Update(_ context.Context, cmd entity.Command) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, c := range m.cmds {
		if c.ID == cmd.ID {
			m.cmds[i] = cmd
			return nil
		}
	}
	return errNotFound
}

func (m *mockCommandRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, c := range m.cmds {
		if c.ID == id {
			m.cmds = append(m.cmds[:i], m.cmds[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockCommandRepo) Reorder(_ context.Context, _ map[entity.StageId][]int64) error {
	return m.reorderErr
}

func (m *mockCommandRepo) ArchiveSession(_ context.Context, sessionID string, _ time.Time) error {
	if m.archiveErr != nil {
		return m.archiveErr
	}
	// Mark commands with matching session_id as having a non-nil archived indicator.
	// For testing purposes, remove them from the in-memory slice.
	var remaining []entity.Command
	for _, c := range m.cmds {
		if c.SessionID != sessionID {
			remaining = append(remaining, c)
		}
	}
	m.cmds = remaining
	return nil
}

// mockBoardRepo is an in-memory BoardRepository for testing.
type mockBoardRepo struct {
	state     entity.BoardState
	exists    bool
	getErr    error
	updateErr error
}

func (m *mockBoardRepo) Exists(_ context.Context) bool { return m.exists }

func (m *mockBoardRepo) Get(_ context.Context) (entity.BoardState, error) {
	if m.getErr != nil {
		return entity.BoardState{}, m.getErr
	}
	return m.state, nil
}

func (m *mockBoardRepo) Update(_ context.Context, b entity.BoardState) (entity.BoardState, error) {
	if m.updateErr != nil {
		return entity.BoardState{}, m.updateErr
	}
	m.state = b
	m.exists = true
	return b, nil
}

// mockSessionRepo is an in-memory SessionRepository for testing.
type mockSessionRepo struct {
	sessions     []entity.Session
	nextID       int64
	activeErr    error
	latestErr    error
	createErr    error
	updateErr    error
	listRangeErr error
}

func (m *mockSessionRepo) GetActive(_ context.Context) (*entity.Session, error) {
	if m.activeErr != nil {
		return nil, m.activeErr
	}
	for i := range m.sessions {
		if m.sessions[i].EndedAt == nil {
			return &m.sessions[i], nil
		}
	}
	return nil, nil
}

func (m *mockSessionRepo) GetLatestByStageId(_ context.Context) (map[entity.StageId]*entity.Session, error) {
	if m.latestErr != nil {
		return nil, m.latestErr
	}
	latest := make(map[entity.StageId]*entity.Session)
	for i := range m.sessions {
		s := &m.sessions[i]
		prev, ok := latest[s.StageId]
		if !ok || s.ID > prev.ID {
			latest[s.StageId] = s
		}
	}
	return latest, nil
}

func (m *mockSessionRepo) Create(_ context.Context, s entity.Session) (entity.Session, error) {
	if m.createErr != nil {
		return entity.Session{}, m.createErr
	}
	m.nextID++
	s.ID = m.nextID
	m.sessions = append(m.sessions, s)
	return s, nil
}

func (m *mockSessionRepo) Update(_ context.Context, s entity.Session) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, existing := range m.sessions {
		if existing.ID == s.ID {
			m.sessions[i] = s
			return nil
		}
	}
	return errNotFound
}

func (m *mockSessionRepo) ListByTimeRange(_ context.Context, _, _ time.Time) ([]entity.Session, error) {
	if m.listRangeErr != nil {
		return nil, m.listRangeErr
	}
	return m.sessions, nil
}

// mockOutcomeRepo is an in-memory OutcomeRepository for testing.
type mockOutcomeRepo struct {
	outcomes  []entity.Outcome
	nextID    int64
	listErr   error
	createErr error
	deleteErr error
}

func (m *mockOutcomeRepo) ListByCommandID(_ context.Context, commandID int64) ([]entity.Outcome, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []entity.Outcome
	for _, o := range m.outcomes {
		if o.CommandID == commandID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (m *mockOutcomeRepo) Create(_ context.Context, o entity.Outcome) (entity.Outcome, error) {
	if m.createErr != nil {
		return entity.Outcome{}, m.createErr
	}
	m.nextID++
	o.ID = m.nextID
	m.outcomes = append(m.outcomes, o)
	return o, nil
}

func (m *mockOutcomeRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, o := range m.outcomes {
		if o.ID == id {
			m.outcomes = append(m.outcomes[:i], m.outcomes[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockSnapshotRepo is an in-memory SnapshotRepository for testing.
type mockSnapshotRepo struct {
	snaps         []entity.Snapshot
	nextID        int64
	listErr       error
	getErr        error
	createErr     error
	updateErr     error
	deleteErr     error
	findByHashErr error
}

func (m *mockSnapshotRepo) List(_ context.Context) ([]entity.Snapshot, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.snaps, nil
}

func (m *mockSnapshotRepo) Get(_ context.Context, id int64) (entity.Snapshot, error) {
	if m.getErr != nil {
		return entity.Snapshot{}, m.getErr
	}
	for _, s := range m.snaps {
		if s.ID == id {
			return s, nil
		}
	}
	return entity.Snapshot{}, errNotFound
}

func (m *mockSnapshotRepo) Create(_ context.Context, s entity.Snapshot) (entity.Snapshot, error) {
	if m.createErr != nil {
		return entity.Snapshot{}, m.createErr
	}
	m.nextID++
	s.ID = m.nextID
	m.snaps = append(m.snaps, s)
	return s, nil
}

func (m *mockSnapshotRepo) Update(_ context.Context, s entity.Snapshot) (entity.Snapshot, error) {
	if m.updateErr != nil {
		return entity.Snapshot{}, m.updateErr
	}
	for i, existing := range m.snaps {
		if existing.ID == s.ID {
			m.snaps[i] = s
			return s, nil
		}
	}
	return entity.Snapshot{}, errNotFound
}

func (m *mockSnapshotRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, s := range m.snaps {
		if s.ID == id {
			m.snaps = append(m.snaps[:i], m.snaps[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockSnapshotRepo) FindByHash(_ context.Context, hash string) (*entity.Snapshot, error) {
	if m.findByHashErr != nil {
		return nil, m.findByHashErr
	}
	for i, s := range m.snaps {
		if s.Hash == hash {
			return &m.snaps[i], nil
		}
	}
	return nil, nil
}

// mockFocusReader is a stub FocusReader for testing.
type mockFocusReader struct {
	result dto.FocusDataDTO
}

func (m *mockFocusReader) GetFocusDataForSessions(_ []FocusSessionWindow) dto.FocusDataDTO {
	return m.result
}
