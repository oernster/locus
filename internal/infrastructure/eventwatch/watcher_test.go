package eventwatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/locus/internal/domain/entity"
)

// writeLine appends a JSON-encoded ClaudeEvent to the file at path.
func writeLine(t *testing.T, path string, ev entity.ClaudeEvent) {
	t.Helper()
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestWatcher_Poll_ReadsNewLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	var got []entity.ClaudeEvent
	w := New(path, func(ev entity.ClaudeEvent) { got = append(got, ev) })

	// No file yet: poll should be a no-op.
	w.Poll()
	if len(got) != 0 {
		t.Fatalf("expected no events before file exists, got %d", len(got))
	}

	// Write first event.
	ev1 := entity.ClaudeEvent{Type: entity.ClaudeEventSessionStart, SessionID: "abc", Timestamp: 1}
	writeLine(t, path, ev1)
	w.Poll()
	if len(got) != 1 {
		t.Fatalf("expected 1 event after first write, got %d", len(got))
	}
	if got[0].Type != entity.ClaudeEventSessionStart || got[0].SessionID != "abc" {
		t.Errorf("unexpected event: %+v", got[0])
	}

	// Second poll with no new content: no-op.
	w.Poll()
	if len(got) != 1 {
		t.Errorf("expected still 1 event on idle poll, got %d", len(got))
	}

	// Write second event.
	ev2 := entity.ClaudeEvent{Type: entity.ClaudeEventToolUse, SessionID: "abc", Tool: "Edit", Target: "x.go", Timestamp: 2}
	writeLine(t, path, ev2)
	w.Poll()
	if len(got) != 2 {
		t.Fatalf("expected 2 events after second write, got %d", len(got))
	}
	if got[1].Tool != "Edit" {
		t.Errorf("unexpected second event: %+v", got[1])
	}
}

func TestWatcher_Poll_SkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	var got []entity.ClaudeEvent
	w := New(path, func(ev entity.ClaudeEvent) { got = append(got, ev) })

	// Write a malformed line followed by a valid line.
	f, _ := os.Create(path)
	f.WriteString("not-json\n")
	f.Close()

	ev := entity.ClaudeEvent{Type: entity.ClaudeEventSessionEnd, SessionID: "z", Timestamp: 3}
	writeLine(t, path, ev)

	w.Poll()

	// Only the valid line should be dispatched.
	if len(got) != 1 {
		t.Fatalf("expected 1 valid event, got %d", len(got))
	}
	if got[0].Type != entity.ClaudeEventSessionEnd {
		t.Errorf("unexpected event: %+v", got[0])
	}
}

func TestWatcher_Poll_EmptyLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	var got []entity.ClaudeEvent
	w := New(path, func(ev entity.ClaudeEvent) { got = append(got, ev) })

	f, _ := os.Create(path)
	f.WriteString("\n\n\n")
	f.Close()

	w.Poll()
	if len(got) != 0 {
		t.Errorf("expected 0 events for empty lines, got %d", len(got))
	}
}

func TestWatcher_Poll_OffsetAdvances(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	count := 0
	w := New(path, func(_ entity.ClaudeEvent) { count++ })

	ev := entity.ClaudeEvent{Type: entity.ClaudeEventSessionStart, SessionID: "s", Timestamp: 1}
	writeLine(t, path, ev)
	w.Poll() // reads 1 event, advances offset

	// Simulate second poll without new content.
	w.Poll()
	if count != 1 {
		t.Errorf("event replayed: count = %d, want 1", count)
	}
}

func TestWatcher_StartStop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	w := New(path, func(_ entity.ClaudeEvent) {})
	w.Start()
	w.Stop()
	// Verify no panic and double-stop is safe.
	w.Stop()
}
