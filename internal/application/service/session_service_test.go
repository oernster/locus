package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestSessionService_GetActive_None(t *testing.T) {
	svc := NewSessionService(&mockSessionRepo{}, &mockCommandRepo{})
	d, err := svc.GetActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Active {
		t.Fatal("expected inactive")
	}
}

func TestSessionService_GetActive_HasActive(t *testing.T) {
	now := time.Now().UTC()
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 10, StageId: entity.StagePlan, StartedAt: now},
		},
	}
	svc := NewSessionService(sessRepo, &mockCommandRepo{})
	d, err := svc.GetActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !d.Active || d.ID != 1 || d.CommandID != 10 {
		t.Fatalf("unexpected DTO: %+v", d)
	}
}

func TestSessionService_GetActive_Error(t *testing.T) {
	sentinel := errors.New("db err")
	svc := NewSessionService(&mockSessionRepo{activeErr: sentinel}, &mockCommandRepo{})
	_, err := svc.GetActive(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_GetLatestByStageId_Empty(t *testing.T) {
	svc := NewSessionService(&mockSessionRepo{}, &mockCommandRepo{})
	m, err := svc.GetLatestByStageId(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %v", m)
	}
}

func TestSessionService_GetLatestByStageId_Populated(t *testing.T) {
	now := time.Now().UTC()
	ended := now.Add(time.Hour)
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now, EndedAt: &ended},
			{ID: 2, CommandID: 2, StageId: entity.StageExecute, StartedAt: now},
		},
	}
	svc := NewSessionService(sessRepo, &mockCommandRepo{})
	m, err := svc.GetLatestByStageId(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["PLAN"]; !ok {
		t.Fatal("PLAN stage missing")
	}
	if _, ok := m["EXECUTE"]; !ok {
		t.Fatal("EXECUTE stage missing")
	}
}

func TestSessionService_GetLatestByStageId_Error(t *testing.T) {
	sentinel := errors.New("db err")
	svc := NewSessionService(&mockSessionRepo{latestErr: sentinel}, &mockCommandRepo{})
	_, err := svc.GetLatestByStageId(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_Start_CommandNotFound(t *testing.T) {
	sentinel := errors.New("not found")
	cmdRepo := &mockCommandRepo{getErr: sentinel}
	svc := NewSessionService(&mockSessionRepo{}, cmdRepo)
	_, err := svc.Start(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionService_Start_GetActiveError(t *testing.T) {
	sentinel := errors.New("active err")
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	sessRepo := &mockSessionRepo{activeErr: sentinel}
	svc := NewSessionService(sessRepo, cmdRepo)
	_, err := svc.Start(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_Start_StopsActiveThenCreates(t *testing.T) {
	now := time.Now().UTC()
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now},
		},
	}
	svc := NewSessionService(sessRepo, cmdRepo)
	d, err := svc.Start(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Active {
		t.Fatal("new session should be active")
	}
	// Old session should now have EndedAt set.
	if sessRepo.sessions[0].EndedAt == nil {
		t.Fatal("old session should be ended")
	}
}

func TestSessionService_Start_StopActiveUpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	now := time.Now().UTC()
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	sessRepo := &mockSessionRepo{
		sessions:  []entity.Session{{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now}},
		updateErr: sentinel,
	}
	svc := NewSessionService(sessRepo, cmdRepo)
	_, err := svc.Start(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_Start_CreateError(t *testing.T) {
	sentinel := errors.New("create err")
	cmdRepo := &mockCommandRepo{
		cmds: []entity.Command{{ID: 1, Title: "T", StageId: entity.StagePlan}},
	}
	sessRepo := &mockSessionRepo{createErr: sentinel}
	svc := NewSessionService(sessRepo, cmdRepo)
	_, err := svc.Start(context.Background(), 1)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_Stop_NoActive(t *testing.T) {
	svc := NewSessionService(&mockSessionRepo{}, &mockCommandRepo{})
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSessionService_Stop_ActiveError(t *testing.T) {
	sentinel := errors.New("active err")
	svc := NewSessionService(&mockSessionRepo{activeErr: sentinel}, &mockCommandRepo{})
	if err := svc.Stop(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestSessionService_Stop_Success(t *testing.T) {
	now := time.Now().UTC()
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now},
		},
	}
	svc := NewSessionService(sessRepo, &mockCommandRepo{})
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sessRepo.sessions[0].EndedAt == nil {
		t.Fatal("session should be ended")
	}
}

func TestSessionService_Stop_UpdateError(t *testing.T) {
	sentinel := errors.New("update err")
	now := time.Now().UTC()
	sessRepo := &mockSessionRepo{
		sessions:  []entity.Session{{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now}},
		updateErr: sentinel,
	}
	svc := NewSessionService(sessRepo, &mockCommandRepo{})
	if err := svc.Stop(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestToSessionDTO_EndedAt(t *testing.T) {
	now := time.Now().UTC()
	ended := now.Add(time.Hour)
	sess := entity.Session{
		ID:        5,
		CommandID: 10,
		StageId:   entity.StageCheck,
		StartedAt: now,
		EndedAt:   &ended,
	}
	d := toSessionDTO(sess)
	if d.Active {
		t.Fatal("should not be active when EndedAt set")
	}
	if d.EndedAt == "" {
		t.Fatal("EndedAt should be set in DTO")
	}
}
