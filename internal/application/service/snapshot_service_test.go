package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func newSnapshotSvc(snapRepo *mockSnapshotRepo, cmdRepo *mockCommandRepo, outcomeRepo *mockOutcomeRepo, boardRepo *mockBoardRepo) *SnapshotService {
	return NewSnapshotService(snapRepo, cmdRepo, outcomeRepo, boardRepo)
}

func populatedRepos() (*mockSnapshotRepo, *mockCommandRepo, *mockOutcomeRepo, *mockBoardRepo) {
	boardRepo := &mockBoardRepo{exists: true, state: entity.BoardState{Name: "Board", UserNamed: true}}
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{
			{ID: 1, Title: "Task A", Status: entity.StatusNotStarted, StageId: entity.StagePlan, CreatedAt: time.Now().UTC()},
		},
	}
	outcomeRepo := &mockOutcomeRepo{}
	snapRepo := &mockSnapshotRepo{}
	return snapRepo, cmdRepo, outcomeRepo, boardRepo
}

func TestSnapshotService_List_Empty(t *testing.T) {
	svc := newSnapshotSvc(&mockSnapshotRepo{}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestSnapshotService_List_Error(t *testing.T) {
	sentinel := errors.New("list err")
	svc := newSnapshotSvc(&mockSnapshotRepo{listErr: sentinel}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	_, err := svc.List(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_List_Populated(t *testing.T) {
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{
			{ID: 1, Name: "snap1", Hash: "abc", SavedAt: time.Now().UTC()},
		},
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "snap1" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestSnapshotService_Save_NewSnapshot(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	d, err := svc.Save(context.Background(), "my-snap")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "my-snap" {
		t.Fatalf("Name = %q, want my-snap", d.Name)
	}
	if len(snapRepo.snaps) != 1 {
		t.Fatal("snapshot should be persisted")
	}
}

func TestSnapshotService_Save_AutoName(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	d, err := svc.Save(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d.Name, "Snapshot ") {
		t.Fatalf("expected auto name, got %q", d.Name)
	}
}

func TestSnapshotService_Save_ExistingHashSameName(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	// First save to get a real hash.
	d1, err := svc.Save(context.Background(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	// Second save with same name;should return existing without creating new.
	d2, err := svc.Save(context.Background(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if d1.ID != d2.ID {
		t.Fatalf("IDs differ: %d vs %d;should reuse existing", d1.ID, d2.ID)
	}
	if len(snapRepo.snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapRepo.snaps))
	}
}

func TestSnapshotService_Save_ExistingHashDifferentName(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	_, err := svc.Save(context.Background(), "old-name")
	if err != nil {
		t.Fatal(err)
	}
	// Same board state, different name;should rename existing.
	d, err := svc.Save(context.Background(), "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "new-name" {
		t.Fatalf("Name = %q, want new-name", d.Name)
	}
	if len(snapRepo.snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapRepo.snaps))
	}
}

func TestSnapshotService_Save_BoardGetError(t *testing.T) {
	sentinel := errors.New("board err")
	boardRepo := &mockBoardRepo{exists: true, getErr: sentinel}
	svc := newSnapshotSvc(&mockSnapshotRepo{}, &mockCommandRepo{}, &mockOutcomeRepo{}, boardRepo)
	_, err := svc.Save(context.Background(), "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Save_CmdListError(t *testing.T) {
	sentinel := errors.New("cmd err")
	boardRepo := &mockBoardRepo{exists: true}
	cmdRepo := &mockCommandRepo{listErr: sentinel}
	svc := newSnapshotSvc(&mockSnapshotRepo{}, cmdRepo, &mockOutcomeRepo{}, boardRepo)
	_, err := svc.Save(context.Background(), "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Save_OutcomeListError(t *testing.T) {
	sentinel := errors.New("outcome err")
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	outcomeRepo.listErr = sentinel
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	_, err := svc.Save(context.Background(), "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Save_FindByHashError(t *testing.T) {
	sentinel := errors.New("find err")
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	snapRepo.findByHashErr = sentinel
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	_, err := svc.Save(context.Background(), "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Save_CreateError(t *testing.T) {
	sentinel := errors.New("create err")
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	snapRepo.createErr = sentinel
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	_, err := svc.Save(context.Background(), "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Save_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	// First save creates a snapshot.
	if _, err := svc.Save(context.Background(), "old"); err != nil {
		t.Fatal(err)
	}
	// Now inject update error and try rename via hash collision.
	snapRepo.updateErr = sentinel
	_, err := svc.Save(context.Background(), "new-name")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Load_NotFound(t *testing.T) {
	svc := newSnapshotSvc(&mockSnapshotRepo{}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	if err := svc.Load(context.Background(), 999); err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestSnapshotService_Load_GetError(t *testing.T) {
	sentinel := errors.New("get err")
	svc := newSnapshotSvc(&mockSnapshotRepo{getErr: sentinel}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	if err := svc.Load(context.Background(), 1); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Load_CorruptData(t *testing.T) {
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "x", Data: "not-json", Hash: "h", SavedAt: time.Now().UTC()}},
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	if err := svc.Load(context.Background(), 1); err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestSnapshotService_Load_Success_CurrentVersion(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	// Save then load.
	saved, err := svc.Save(context.Background(), "snap1")
	if err != nil {
		t.Fatal(err)
	}
	// Clear commands.
	cmdRepo.cmds = nil
	if err := svc.Load(context.Background(), saved.ID); err != nil {
		t.Fatal(err)
	}
	if len(cmdRepo.cmds) == 0 {
		t.Fatal("commands should be restored after load")
	}
}

func TestSnapshotService_Load_MigrationOldStages(t *testing.T) {
	// Build snapshot JSON with old stage IDs.
	oldJSON := `{"version":4,"board":{"Name":"","UserNamed":false,"StageLabels":{"DESIGN":"Planning"}},"commands":[{"ID":1,"Title":"T","Status":"Not Started","StageId":"BUILD","SortIndex":0,"CreatedAt":"0001-01-01T00:00:00Z"}],"outcomes":[]}`
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "old", Data: oldJSON, Hash: "h", SavedAt: time.Now().UTC()}},
	}
	cmdRepo := &mockCommandRepo{}
	boardRepo := &mockBoardRepo{exists: true}
	svc := newSnapshotSvc(snapRepo, cmdRepo, &mockOutcomeRepo{}, boardRepo)
	if err := svc.Load(context.Background(), 1); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	// Command should have migrated BUILD -> EXECUTE.
	if len(cmdRepo.cmds) == 0 {
		t.Fatal("commands not restored")
	}
	if cmdRepo.cmds[0].StageId != entity.StageExecute {
		t.Fatalf("StageId = %q, want EXECUTE", cmdRepo.cmds[0].StageId)
	}
	// Stage label DESIGN should be migrated to PLAN.
	if boardRepo.state.StageLabels["PLAN"] != "Planning" {
		t.Fatalf("label not migrated: %v", boardRepo.state.StageLabels)
	}
}

func TestSnapshotService_Load_MigrationLabelKeyNotInAliases(t *testing.T) {
	// Stage label key "PLAN" is not in stageAliases;should be kept unchanged.
	oldJSON := `{"version":4,"board":{"Name":"","UserNamed":false,"StageLabels":{"PLAN":"Backlog","DESIGN":"Planning"}},"commands":[],"outcomes":[]}`
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "old", Data: oldJSON, Hash: "h", SavedAt: time.Now().UTC()}},
	}
	cmdRepo := &mockCommandRepo{}
	boardRepo := &mockBoardRepo{exists: true}
	svc := newSnapshotSvc(snapRepo, cmdRepo, &mockOutcomeRepo{}, boardRepo)
	if err := svc.Load(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "PLAN" key not in aliases;kept; "DESIGN" migrated to "PLAN".
	if boardRepo.state.StageLabels["PLAN"] == "" {
		t.Fatalf("PLAN label should be present: %v", boardRepo.state.StageLabels)
	}
}

func TestSnapshotService_Load_CmdListError(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	cmdRepo.listErr = errors.New("list err")
	if err := svc.Load(context.Background(), saved.ID); err == nil {
		t.Fatal("expected error on list commands")
	}
}

func TestSnapshotService_Load_CmdDeleteError(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	cmdRepo.deleteErr = errors.New("delete err")
	if err := svc.Load(context.Background(), saved.ID); err == nil {
		t.Fatal("expected error on delete command")
	}
}

func TestSnapshotService_Load_BoardUpdateError(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	cmdRepo.cmds = nil
	boardRepo.updateErr = errors.New("board err")
	if err := svc.Load(context.Background(), saved.ID); err == nil {
		t.Fatal("expected error on board update")
	}
}

func TestSnapshotService_Load_CmdCreateError(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	cmdRepo.cmds = nil
	boardRepo.updateErr = nil
	cmdRepo.createErr = errors.New("create err")
	if err := svc.Load(context.Background(), saved.ID); err == nil {
		t.Fatal("expected error on create command")
	}
}

func TestSnapshotService_Load_OutcomeWithOrphan(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	// Add an outcome for the command.
	outcomeRepo.outcomes = []entity.Outcome{
		{ID: 1, CommandID: 1, Note: "note"},
	}
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	// Clear state for load.
	cmdRepo.cmds = nil
	outcomeRepo.outcomes = nil
	if err := svc.Load(context.Background(), saved.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotService_Load_OutcomeCreateError(t *testing.T) {
	snapRepo, cmdRepo, outcomeRepo, boardRepo := populatedRepos()
	// Seed an outcome for the command.
	outcomeRepo.outcomes = []entity.Outcome{
		{ID: 1, CommandID: 1, Note: "outcome"},
	}
	svc := newSnapshotSvc(snapRepo, cmdRepo, outcomeRepo, boardRepo)
	saved, err := svc.Save(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	cmdRepo.cmds = nil
	outcomeRepo.outcomes = nil
	// Inject error on create after commands are restored.
	outcomeRepo.createErr = errors.New("outcome create err")
	if err := svc.Load(context.Background(), saved.ID); err == nil {
		t.Fatal("expected error on outcome create")
	}
}

func TestSnapshotService_Delete_Success(t *testing.T) {
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "x", Hash: "h", SavedAt: time.Now().UTC()}},
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(snapRepo.snaps) != 0 {
		t.Fatal("snapshot should be deleted")
	}
}

func TestSnapshotService_Delete_Error(t *testing.T) {
	sentinel := errors.New("delete err")
	svc := newSnapshotSvc(&mockSnapshotRepo{deleteErr: sentinel}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	if err := svc.Delete(context.Background(), 1); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Rename_Success(t *testing.T) {
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "old", Hash: "h", SavedAt: time.Now().UTC()}},
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	d, err := svc.Rename(context.Background(), 1, "  new  ")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "new" {
		t.Fatalf("Name = %q, want 'new'", d.Name)
	}
}

func TestSnapshotService_Rename_EmptyName(t *testing.T) {
	snapRepo := &mockSnapshotRepo{
		snaps: []entity.Snapshot{{ID: 1, Name: "old", Hash: "h", SavedAt: time.Now().UTC()}},
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	_, err := svc.Rename(context.Background(), 1, "   ")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSnapshotService_Rename_GetError(t *testing.T) {
	sentinel := errors.New("get err")
	svc := newSnapshotSvc(&mockSnapshotRepo{getErr: sentinel}, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	_, err := svc.Rename(context.Background(), 1, "x")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSnapshotService_Rename_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	snapRepo := &mockSnapshotRepo{
		snaps:     []entity.Snapshot{{ID: 1, Name: "old", Hash: "h", SavedAt: time.Now().UTC()}},
		updateErr: sentinel,
	}
	svc := newSnapshotSvc(snapRepo, &mockCommandRepo{}, &mockOutcomeRepo{}, &mockBoardRepo{exists: true})
	_, err := svc.Rename(context.Background(), 1, "new")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
