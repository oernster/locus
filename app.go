package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/application/service"
	"github.com/oernster/locus/internal/infrastructure/eventwatch"
)

// App is the Wails application struct. All exported methods are bound to the
// frontend via Wails IPC.
type App struct {
	ctx         context.Context
	commandSvc  *service.CommandService
	sessionSvc  *service.SessionService
	outcomeSvc  *service.OutcomeService
	boardSvc    *service.BoardService
	snapshotSvc *service.SnapshotService
	focusSvc    *service.FocusService
	claudeSvc   *service.ClaudeSessionService
	watcher     *eventwatch.Watcher
	// boardNotify receives a signal whenever a dynamic board item is created,
	// advanced, or archived. The startup goroutine forwards these to the frontend
	// as a "locus:board-updated" Wails event.
	boardNotify <-chan struct{}
}

// NewApp creates an App with all services wired in.
func NewApp(
	commandSvc *service.CommandService,
	sessionSvc *service.SessionService,
	outcomeSvc *service.OutcomeService,
	boardSvc *service.BoardService,
	snapshotSvc *service.SnapshotService,
	focusSvc *service.FocusService,
	claudeSvc *service.ClaudeSessionService,
	watcher *eventwatch.Watcher,
	boardNotify <-chan struct{},
) *App {
	return &App{
		commandSvc:  commandSvc,
		sessionSvc:  sessionSvc,
		outcomeSvc:  outcomeSvc,
		boardSvc:    boardSvc,
		snapshotSvc: snapshotSvc,
		focusSvc:    focusSvc,
		claudeSvc:   claudeSvc,
		watcher:     watcher,
		boardNotify: boardNotify,
	}
}

// startup is called when the app starts. The context is saved, the event watcher
// is started, and a goroutine forwards board-update notifications to the frontend.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.watcher != nil {
		a.watcher.Start()
	}
	// Forward dynamic-item notifications to the React frontend.
	go func() {
		for range a.boardNotify {
			runtime.EventsEmit(ctx, "locus:board-updated")
		}
	}()
}

// --- Command methods ---

// ListCommands returns all commands, optionally filtered by stageId.
func (a *App) ListCommands(stageId string) ([]dto.CommandDTO, error) {
	return a.commandSvc.List(a.ctx, stageId)
}

// CreateCommand adds a new command.
func (a *App) CreateCommand(title, stageId string) (dto.CommandDTO, error) {
	return a.commandSvc.Create(a.ctx, title, stageId)
}

// UpdateCommand modifies an existing command.
func (a *App) UpdateCommand(id int64, title, status, stageId string) (dto.CommandDTO, error) {
	return a.commandSvc.Update(a.ctx, id, title, status, stageId)
}

// DeleteCommand removes a command.
func (a *App) DeleteCommand(id int64) error {
	return a.commandSvc.Delete(a.ctx, id)
}

// ReorderCommands persists a new sort order for commands across stages.
func (a *App) ReorderCommands(byStageId map[string][]int64) error {
	return a.commandSvc.Reorder(a.ctx, byStageId)
}

// --- Board methods ---

// GetBoard returns the current board state.
func (a *App) GetBoard() (dto.BoardDTO, error) {
	return a.boardSvc.Get(a.ctx)
}

// UpdateBoard renames the board.
func (a *App) UpdateBoard(name string) (dto.BoardDTO, error) {
	return a.boardSvc.UpdateName(a.ctx, name)
}

// UpdateStageLabels replaces stage label overrides.
func (a *App) UpdateStageLabels(labels map[string]string) (dto.BoardDTO, error) {
	return a.boardSvc.UpdateStageLabels(a.ctx, labels)
}

// ResetBoard clears all commands and resets the board.
func (a *App) ResetBoard() error {
	return a.boardSvc.Reset(a.ctx)
}

// --- Session methods ---

// GetActiveSession returns the currently active session, or an inactive DTO.
func (a *App) GetActiveSession() (dto.SessionDTO, error) {
	return a.sessionSvc.GetActive(a.ctx)
}

// StartSession begins a session for the given command.
func (a *App) StartSession(commandId int64) (dto.SessionDTO, error) {
	return a.sessionSvc.Start(a.ctx, commandId)
}

// StopSession ends the currently active session.
func (a *App) StopSession() error {
	return a.sessionSvc.Stop(a.ctx)
}

// GetLatestSessionsByStageId returns the most-recent session per stage.
func (a *App) GetLatestSessionsByStageId() (map[string]dto.SessionDTO, error) {
	return a.sessionSvc.GetLatestByStageId(a.ctx)
}

// --- Outcome methods ---

// ListOutcomes returns all outcomes for a command.
func (a *App) ListOutcomes(commandId int64) ([]dto.OutcomeDTO, error) {
	return a.outcomeSvc.ListByCommand(a.ctx, commandId)
}

// CreateOutcome adds a new outcome to a command.
func (a *App) CreateOutcome(commandId int64, note string) (dto.OutcomeDTO, error) {
	return a.outcomeSvc.Create(a.ctx, commandId, note)
}

// DeleteOutcome removes an outcome.
func (a *App) DeleteOutcome(outcomeId int64) error {
	return a.outcomeSvc.Delete(a.ctx, outcomeId)
}

// --- Snapshot methods ---

// ListSnapshots returns all snapshot summaries.
func (a *App) ListSnapshots() ([]dto.SnapshotDTO, error) {
	return a.snapshotSvc.List(a.ctx)
}

// SaveSnapshot serialises the current board to a named snapshot.
func (a *App) SaveSnapshot(name string) (dto.SnapshotDTO, error) {
	return a.snapshotSvc.Save(a.ctx, name)
}

// LoadSnapshot restores the board from a snapshot.
func (a *App) LoadSnapshot(snapshotId int64) error {
	return a.snapshotSvc.Load(a.ctx, snapshotId)
}

// DeleteSnapshot removes a snapshot.
func (a *App) DeleteSnapshot(snapshotId int64) error {
	return a.snapshotSvc.Delete(a.ctx, snapshotId)
}

// RenameSnapshot changes the display name of a snapshot.
func (a *App) RenameSnapshot(snapshotId int64, name string) (dto.SnapshotDTO, error) {
	return a.snapshotSvc.Rename(a.ctx, snapshotId, name)
}

// --- Focus methods ---

// GetFocusData returns focus-reader data for a board stage.
func (a *App) GetFocusData(stageId string) (dto.FocusDataDTO, error) {
	return a.focusSvc.GetFocusDataForStage(a.ctx, stageId)
}

// GetFocusDataForTimeRange returns aggregated focus data for the supplied Unix
// second time range. The frontend computes calendar boundaries in local time.
func (a *App) GetFocusDataForTimeRange(startUnix, endUnix int64) (dto.FocusDataDTO, error) {
	return a.focusSvc.GetFocusDataForTimeRange(a.ctx, startUnix, endUnix)
}
