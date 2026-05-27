package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

func makeSnap(name, data, hash string) entity.Snapshot {
	return entity.Snapshot{Name: name, Data: data, Hash: hash, SavedAt: time.Now().UTC()}
}

func TestSQLiteSnapshotRepository_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	s := makeSnap("snap1", `{"v":1}`, "abc123")
	created, err := repo.Create(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("ID not assigned")
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "snap1" || got.Hash != "abc123" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestSQLiteSnapshotRepository_Create_ZeroSavedAt(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	s := entity.Snapshot{Name: "x", Data: "{}", Hash: "h"}
	created, err := repo.Create(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	if created.SavedAt.IsZero() {
		t.Fatal("SavedAt should be defaulted")
	}
}

func TestSQLiteSnapshotRepository_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	_, err := repo.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestSQLiteSnapshotRepository_List_OrderedDesc(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	s1 := entity.Snapshot{Name: "old", Data: "{}", Hash: "h1", SavedAt: now}
	s2 := entity.Snapshot{Name: "new", Data: "{}", Hash: "h2", SavedAt: now.Add(time.Second)}
	_, _ = repo.Create(ctx, s1)
	_, _ = repo.Create(ctx, s2)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].Name != "new" {
		t.Fatalf("expected DESC order, first = %q", list[0].Name)
	}
}

func TestSQLiteSnapshotRepository_Update(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, makeSnap("orig", "{}", "h1"))
	created.Name = "renamed"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "renamed" {
		t.Fatalf("Name = %q, want renamed", updated.Name)
	}
	got, _ := repo.Get(ctx, created.ID)
	if got.Name != "renamed" {
		t.Fatalf("not persisted: %q", got.Name)
	}
}

func TestSQLiteSnapshotRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, makeSnap("x", "{}", "h"))
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	_, err := repo.Get(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestSQLiteSnapshotRepository_FindByHash_Found(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, makeSnap("s", `{}`, "myhash"))
	found, err := repo.FindByHash(ctx, "myhash")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != created.ID {
		t.Fatalf("unexpected: %v", found)
	}
}

func TestSQLiteSnapshotRepository_FindByHash_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteSnapshotRepository(db)
	found, err := repo.FindByHash(context.Background(), "nohash")
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected nil for missing hash")
	}
}
