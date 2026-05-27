package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestBoardService_Get_BoardNotExists_CreatesDefault(t *testing.T) {
	boardRepo := &mockBoardRepo{exists: false}
	cmdRepo := &mockCommandRepo{}
	svc := NewBoardService(boardRepo, cmdRepo)

	d, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !boardRepo.exists {
		t.Fatal("board should be created")
	}
	if !d.IsNewUnnamed {
		t.Fatal("expected IsNewUnnamed=true for fresh board")
	}
	if !d.IsEmpty {
		t.Fatal("expected IsEmpty=true for fresh board")
	}
}

func TestBoardService_Get_BoardNotExists_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	boardRepo := &mockBoardRepo{exists: false, updateErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.Get(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_Get_BoardExists(t *testing.T) {
	boardRepo := &mockBoardRepo{
		exists: true,
		state:  entity.BoardState{Name: "My Board", UserNamed: true},
	}
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	svc := NewBoardService(boardRepo, cmdRepo)
	d, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "My Board" || !d.UserNamed {
		t.Fatalf("unexpected board: %+v", d)
	}
	if d.IsEmpty {
		t.Fatal("board should not be empty")
	}
}

func TestBoardService_Get_GetError(t *testing.T) {
	sentinel := errors.New("get err")
	boardRepo := &mockBoardRepo{exists: true, getErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.Get(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_Get_IsEmptyError(t *testing.T) {
	sentinel := errors.New("list err")
	boardRepo := &mockBoardRepo{exists: true}
	cmdRepo := &mockCommandRepo{listErr: sentinel}
	svc := NewBoardService(boardRepo, cmdRepo)
	_, err := svc.Get(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateName_Success(t *testing.T) {
	boardRepo := &mockBoardRepo{exists: true, state: entity.BoardState{Name: "Old"}}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	d, err := svc.UpdateName(context.Background(), "New Name")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "New Name" || !d.UserNamed {
		t.Fatalf("unexpected board: %+v", d)
	}
}

func TestBoardService_UpdateName_GetError(t *testing.T) {
	sentinel := errors.New("get err")
	boardRepo := &mockBoardRepo{exists: true, getErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.UpdateName(context.Background(), "X")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateName_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	boardRepo := &mockBoardRepo{exists: true, updateErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.UpdateName(context.Background(), "X")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateName_IsEmptyError(t *testing.T) {
	sentinel := errors.New("list err")
	boardRepo := &mockBoardRepo{exists: true}
	cmdRepo := &mockCommandRepo{listErr: sentinel}
	svc := NewBoardService(boardRepo, cmdRepo)
	_, err := svc.UpdateName(context.Background(), "X")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateStageLabels_Success(t *testing.T) {
	boardRepo := &mockBoardRepo{exists: true}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	labels := map[string]string{"PLAN": "Backlog"}
	d, err := svc.UpdateStageLabels(context.Background(), labels)
	if err != nil {
		t.Fatal(err)
	}
	if d.StageLabels["PLAN"] != "Backlog" {
		t.Fatalf("unexpected labels: %v", d.StageLabels)
	}
}

func TestBoardService_UpdateStageLabels_GetError(t *testing.T) {
	sentinel := errors.New("get err")
	boardRepo := &mockBoardRepo{exists: true, getErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.UpdateStageLabels(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateStageLabels_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	boardRepo := &mockBoardRepo{exists: true, updateErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	_, err := svc.UpdateStageLabels(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_UpdateStageLabels_IsEmptyError(t *testing.T) {
	sentinel := errors.New("list err")
	boardRepo := &mockBoardRepo{exists: true}
	cmdRepo := &mockCommandRepo{listErr: sentinel}
	svc := NewBoardService(boardRepo, cmdRepo)
	_, err := svc.UpdateStageLabels(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_Reset_Success(t *testing.T) {
	boardRepo := &mockBoardRepo{exists: true, state: entity.BoardState{Name: "X"}}
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{
			{ID: 1, Title: "T1", StageId: entity.StagePlan},
			{ID: 2, Title: "T2", StageId: entity.StageExecute},
		},
	}
	svc := NewBoardService(boardRepo, cmdRepo)
	if err := svc.Reset(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(cmdRepo.cmds) != 0 {
		t.Fatal("commands should be deleted after reset")
	}
	if boardRepo.state.Name != "" || boardRepo.state.UserNamed {
		t.Fatalf("board should be reset: %+v", boardRepo.state)
	}
}

func TestBoardService_Reset_ListError(t *testing.T) {
	sentinel := errors.New("list err")
	svc := NewBoardService(&mockBoardRepo{exists: true}, &mockCommandRepo{listErr: sentinel})
	if err := svc.Reset(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_Reset_DeleteError(t *testing.T) {
	sentinel := errors.New("delete err")
	boardRepo := &mockBoardRepo{exists: true}
	cmdRepo := &mockCommandRepo{
		cmds:      []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
		deleteErr: sentinel,
	}
	svc := NewBoardService(boardRepo, cmdRepo)
	if err := svc.Reset(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_Reset_BoardUpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	boardRepo := &mockBoardRepo{exists: true, updateErr: sentinel}
	svc := NewBoardService(boardRepo, &mockCommandRepo{})
	if err := svc.Reset(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestBoardService_IsEmpty_True(t *testing.T) {
	svc := NewBoardService(&mockBoardRepo{exists: true}, &mockCommandRepo{})
	empty, err := svc.IsEmpty(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("expected empty")
	}
}

func TestBoardService_IsEmpty_False(t *testing.T) {
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	svc := NewBoardService(&mockBoardRepo{exists: true}, cmdRepo)
	empty, err := svc.IsEmpty(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected not empty")
	}
}

func TestBoardService_IsEmpty_Error(t *testing.T) {
	sentinel := errors.New("list err")
	svc := NewBoardService(&mockBoardRepo{exists: true}, &mockCommandRepo{listErr: sentinel})
	_, err := svc.IsEmpty(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
