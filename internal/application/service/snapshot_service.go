package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
	"github.com/oernster/locus/internal/domain/repository"
)

// snapshotSchemaVersion is incremented when the snapshot JSON format changes.
const snapshotSchemaVersion = 5

// snapshotData is the serialised form of a board snapshot.
type snapshotData struct {
	Version  int                `json:"version"`
	Board    entity.BoardState  `json:"board"`
	Commands []entity.Command   `json:"commands"`
	Outcomes []entity.Outcome   `json:"outcomes"`
}

// stageAliases maps old stage IDs (pre-version-5) to current ones.
var stageAliases = map[string]string{
	"DESIGN":   "PLAN",
	"BUILD":    "EXECUTE",
	"REVIEW":   "CHECK",
	"COMPLETE": "DONE",
}

// SnapshotService handles snapshot save/load operations.
type SnapshotService struct {
	snapshotRepo repository.SnapshotRepository
	commandRepo  repository.CommandRepository
	outcomeRepo  repository.OutcomeRepository
	boardRepo    repository.BoardRepository
}

// NewSnapshotService creates a SnapshotService.
func NewSnapshotService(
	snapshotRepo repository.SnapshotRepository,
	commandRepo repository.CommandRepository,
	outcomeRepo repository.OutcomeRepository,
	boardRepo repository.BoardRepository,
) *SnapshotService {
	return &SnapshotService{
		snapshotRepo: snapshotRepo,
		commandRepo:  commandRepo,
		outcomeRepo:  outcomeRepo,
		boardRepo:    boardRepo,
	}
}

// List returns all snapshot summaries ordered by most-recent first.
func (s *SnapshotService) List(ctx context.Context) ([]dto.SnapshotDTO, error) {
	snaps, err := s.snapshotRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SnapshotDTO, len(snaps))
	for i, sn := range snaps {
		out[i] = toSnapshotDTO(sn)
	}
	return out, nil
}

// Save serialises the current board state to a named snapshot.
// If the board hash already exists and the name differs, the existing entry is
// renamed; otherwise a new snapshot is created.
func (s *SnapshotService) Save(ctx context.Context, name string) (dto.SnapshotDTO, error) {
	data, hash, err := s.serialise(ctx)
	if err != nil {
		return dto.SnapshotDTO{}, err
	}

	autoName := name
	if autoName == "" {
		autoName = fmt.Sprintf("Snapshot %s", time.Now().UTC().Format("2006-01-02 15:04"))
	}

	existing, err := s.snapshotRepo.FindByHash(ctx, hash)
	if err != nil {
		return dto.SnapshotDTO{}, err
	}
	if existing != nil {
		if existing.Name != autoName {
			existing.Name = autoName
			updated, uerr := s.snapshotRepo.Update(ctx, *existing)
			if uerr != nil {
				return dto.SnapshotDTO{}, uerr
			}
			return toSnapshotDTO(updated), nil
		}
		return toSnapshotDTO(*existing), nil
	}

	snap := entity.Snapshot{
		Name:    autoName,
		Data:    data,
		Hash:    hash,
		SavedAt: time.Now().UTC(),
	}
	created, err := s.snapshotRepo.Create(ctx, snap)
	if err != nil {
		return dto.SnapshotDTO{}, err
	}
	return toSnapshotDTO(created), nil
}

// Load restores the board from a snapshot, migrating older stage IDs as needed.
func (s *SnapshotService) Load(ctx context.Context, id int64) error {
	snap, err := s.snapshotRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	var data snapshotData
	if err := json.Unmarshal([]byte(snap.Data), &data); err != nil {
		return fmt.Errorf("corrupt snapshot: %w", err)
	}

	// Migrate stage IDs from older schema versions.
	if data.Version < snapshotSchemaVersion {
		for i, c := range data.Commands {
			if alias, ok := stageAliases[string(c.StageId)]; ok {
				data.Commands[i].StageId = entity.StageId(alias)
			}
		}
		if data.Board.StageLabels != nil {
			migrated := make(map[string]string, len(data.Board.StageLabels))
			for k, v := range data.Board.StageLabels {
				if alias, ok := stageAliases[k]; ok {
					migrated[alias] = v
				} else {
					migrated[k] = v
				}
			}
			data.Board.StageLabels = migrated
		}
	}

	// Clear current board.
	existing, err := s.commandRepo.List(ctx, nil)
	if err != nil {
		return err
	}
	for _, c := range existing {
		if err := s.commandRepo.Delete(ctx, c.ID); err != nil {
			return err
		}
	}

	// Restore board state.
	if _, err := s.boardRepo.Update(ctx, data.Board); err != nil {
		return err
	}

	// Restore commands and outcomes, remapping IDs.
	idMap := make(map[int64]int64, len(data.Commands))
	for _, c := range data.Commands {
		oldID := c.ID
		c.ID = 0
		created, err := s.commandRepo.Create(ctx, c)
		if err != nil {
			return err
		}
		idMap[oldID] = created.ID
	}

	for _, o := range data.Outcomes {
		o.ID = 0
		o.CommandID = idMap[o.CommandID]
		if o.CommandID == 0 {
			continue // orphaned outcome
		}
		if _, err := s.outcomeRepo.Create(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a snapshot.
func (s *SnapshotService) Delete(ctx context.Context, id int64) error {
	return s.snapshotRepo.Delete(ctx, id)
}

// Rename changes the display name of a snapshot.
func (s *SnapshotService) Rename(ctx context.Context, id int64, name string) (dto.SnapshotDTO, error) {
	snap, err := s.snapshotRepo.Get(ctx, id)
	if err != nil {
		return dto.SnapshotDTO{}, err
	}
	snap.Name = strings.TrimSpace(name)
	if snap.Name == "" {
		return dto.SnapshotDTO{}, fmt.Errorf("name must not be empty")
	}
	updated, err := s.snapshotRepo.Update(ctx, snap)
	if err != nil {
		return dto.SnapshotDTO{}, err
	}
	return toSnapshotDTO(updated), nil
}

func (s *SnapshotService) serialise(ctx context.Context) (string, string, error) {
	board, err := s.boardRepo.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cmds, err := s.commandRepo.List(ctx, nil)
	if err != nil {
		return "", "", err
	}

	var outcomes []entity.Outcome
	for _, c := range cmds {
		oo, err := s.outcomeRepo.ListByCommandID(ctx, c.ID)
		if err != nil {
			return "", "", err
		}
		outcomes = append(outcomes, oo...)
	}

	data := snapshotData{
		Version:  snapshotSchemaVersion,
		Board:    board,
		Commands: cmds,
		Outcomes: outcomes,
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(raw)
	return string(raw), fmt.Sprintf("%x", sum), nil
}

func toSnapshotDTO(s entity.Snapshot) dto.SnapshotDTO {
	return dto.SnapshotDTO{
		ID:      s.ID,
		Name:    s.Name,
		SavedAt: s.SavedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
