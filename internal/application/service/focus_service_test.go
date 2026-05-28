package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/domain/entity"
)

func TestFocusService_GetFocusDataForTimeRange(t *testing.T) {
	want := dto.FocusDataDTO{
		Available:    true,
		TotalSeconds: 3600,
	}
	reader := &mockFocusReader{result: want}
	svc := NewFocusService(&mockSessionRepo{}, reader)

	got, err := svc.GetFocusDataForTimeRange(context.Background(), 1000, 4600)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalSeconds != want.TotalSeconds {
		t.Fatalf("TotalSeconds = %d, want %d", got.TotalSeconds, want.TotalSeconds)
	}
}

func TestFocusService_GetFocusDataForStage_RepoError(t *testing.T) {
	sentinel := errors.New("db err")
	svc := NewFocusService(&mockSessionRepo{listRangeErr: sentinel}, &mockFocusReader{})
	d, err := svc.GetFocusDataForStage(context.Background(), "PLAN")
	if err != nil {
		t.Fatal("should not return error on repo error, returns available:false")
	}
	if d.Available {
		t.Fatal("expected Available=false on error")
	}
}

func TestFocusService_GetFocusDataForStage_NoSessions_RollingWindow(t *testing.T) {
	want := dto.FocusDataDTO{Available: true, TotalSeconds: 500}
	reader := &mockFocusReader{result: want}
	// No sessions in mock repo.
	svc := NewFocusService(&mockSessionRepo{}, reader)
	d, err := svc.GetFocusDataForStage(context.Background(), "EXECUTE")
	if err != nil {
		t.Fatal(err)
	}
	// Reader called with rolling window; returns reader's result.
	if d.TotalSeconds != 500 {
		t.Fatalf("TotalSeconds = %d, want 500", d.TotalSeconds)
	}
	if d.StageId != "EXECUTE" {
		t.Fatalf("StageId = %q, want EXECUTE", d.StageId)
	}
}

func TestFocusService_GetFocusDataForStage_WithSessions(t *testing.T) {
	now := time.Now().UTC()
	ended := now.Add(time.Hour)
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now, EndedAt: &ended},
		},
	}
	want := dto.FocusDataDTO{Available: true, TotalSeconds: 3600}
	reader := &mockFocusReader{result: want}
	svc := NewFocusService(sessRepo, reader)

	d, err := svc.GetFocusDataForStage(context.Background(), "PLAN")
	if err != nil {
		t.Fatal(err)
	}
	if d.TotalSeconds != 3600 {
		t.Fatalf("TotalSeconds = %d, want 3600", d.TotalSeconds)
	}
	if d.StageId != "PLAN" {
		t.Fatalf("StageId = %q, want PLAN", d.StageId)
	}
}

func TestFocusService_GetFocusDataForStage_ActiveSession_NilEndedAt(t *testing.T) {
	now := time.Now().UTC()
	// Active session with nil EndedAt;should use now as end.
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StageExecute, StartedAt: now, EndedAt: nil},
		},
	}
	reader := &mockFocusReader{result: dto.FocusDataDTO{Available: true}}
	svc := NewFocusService(sessRepo, reader)
	_, err := svc.GetFocusDataForStage(context.Background(), "EXECUTE")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFocusService_GetFocusDataForStage_WrongStageSessions(t *testing.T) {
	now := time.Now().UTC()
	ended := now.Add(time.Hour)
	// Sessions exist, but for a different stage;causes rolling window.
	sessRepo := &mockSessionRepo{
		sessions: []entity.Session{
			{ID: 1, CommandID: 1, StageId: entity.StagePlan, StartedAt: now, EndedAt: &ended},
		},
	}
	want := dto.FocusDataDTO{Available: true}
	reader := &mockFocusReader{result: want}
	svc := NewFocusService(sessRepo, reader)
	d, err := svc.GetFocusDataForStage(context.Background(), "EXECUTE")
	if err != nil {
		t.Fatal(err)
	}
	if d.StageId != "EXECUTE" {
		t.Fatalf("StageId = %q, want EXECUTE", d.StageId)
	}
}
