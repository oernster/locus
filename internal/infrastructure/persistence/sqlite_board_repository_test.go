package persistence

import (
	"context"
	"testing"

	"github.com/oernster/locus/internal/domain/entity"
)

func TestSQLiteBoardRepository_ExistsAndGet_Empty(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteBoardRepository(db)
	ctx := context.Background()

	if repo.Exists(ctx) {
		t.Fatal("board should not exist in fresh DB")
	}
	b, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "" || b.UserNamed {
		t.Fatalf("unexpected board state: %+v", b)
	}
}

func TestSQLiteBoardRepository_Update_Insert(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteBoardRepository(db)
	ctx := context.Background()

	bs := entity.BoardState{Name: "My Board", UserNamed: true, StageLabels: map[string]string{"PLAN": "Backlog"}}
	updated, err := repo.Update(ctx, bs)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "My Board" || !updated.UserNamed {
		t.Fatalf("unexpected: %+v", updated)
	}
	if !repo.Exists(ctx) {
		t.Fatal("board should exist after update")
	}
}

func TestSQLiteBoardRepository_Update_Upsert(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteBoardRepository(db)
	ctx := context.Background()

	_, _ = repo.Update(ctx, entity.BoardState{Name: "Old"})
	updated, err := repo.Update(ctx, entity.BoardState{Name: "New", UserNamed: true})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "New" || !updated.UserNamed {
		t.Fatalf("upsert failed: %+v", updated)
	}
}

func TestSQLiteBoardRepository_Get_WithLabels(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteBoardRepository(db)
	ctx := context.Background()

	labels := map[string]string{"EXECUTE": "Doing"}
	_, _ = repo.Update(ctx, entity.BoardState{Name: "B", UserNamed: true, StageLabels: labels})
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.StageLabels["EXECUTE"] != "Doing" {
		t.Fatalf("labels not persisted: %v", got.StageLabels)
	}
}

func TestSQLiteBoardRepository_Get_EmptyLabels(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteBoardRepository(db)
	ctx := context.Background()

	_, _ = repo.Update(ctx, entity.BoardState{Name: "B"})
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.StageLabels != nil {
		t.Fatalf("expected nil labels, got %v", got.StageLabels)
	}
}

func TestEncodeDecodeLabels_RoundTrip(t *testing.T) {
	labels := map[string]string{"PLAN": "Backlog", "DONE": "Released"}
	encoded, err := encodeLabels(labels)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeLabels(encoded)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range labels {
		if decoded[k] != v {
			t.Errorf("labels[%q] = %q, want %q", k, decoded[k], v)
		}
	}
}

func TestEncodeLabels_Nil(t *testing.T) {
	s, err := encodeLabels(nil)
	if err != nil {
		t.Fatal(err)
	}
	if s != "{}" {
		t.Fatalf("encodeLabels(nil) = %q, want {}", s)
	}
}

func TestDecodeLabels_EmptyString(t *testing.T) {
	m, err := decodeLabels("")
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatalf("expected nil, got %v", m)
	}
}

func TestDecodeLabels_EmptyObject(t *testing.T) {
	m, err := decodeLabels("{}")
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatalf("expected nil for {}, got %v", m)
	}
}

func TestDecodeLabels_InvalidJSON(t *testing.T) {
	_, err := decodeLabels("not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
