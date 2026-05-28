package eventwatch

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/oernster/locus/internal/domain/entity"
)

const pollInterval = 500 * time.Millisecond

// Watcher polls a JSONL sidecar file and emits ClaudeEvents to a handler.
// New lines appended to the file since the last poll are parsed and dispatched.
type Watcher struct {
	path     string
	offset   int64
	ticker   *time.Ticker
	handler  func(entity.ClaudeEvent)
	done     chan struct{}
	tickerFn func() <-chan time.Time // injectable for tests
}

// New creates a Watcher for the given file path.
func New(path string, handler func(entity.ClaudeEvent)) *Watcher {
	return &Watcher{
		path:    path,
		handler: handler,
	}
}

// init seeks to the current end of the file so only new events are processed.
func (w *Watcher) init() {
	f, err := os.Open(w.path)
	if err != nil {
		return // File may not exist yet; offset stays 0.
	}
	defer f.Close()
	end, err := f.Seek(0, io.SeekEnd)
	if err == nil {
		w.offset = end
	}
}

// Start begins polling in a background goroutine.
func (w *Watcher) Start() {
	w.init()
	w.done = make(chan struct{})
	w.ticker = time.NewTicker(pollInterval)
	ch := w.ticker.C
	if w.tickerFn != nil {
		ch = w.tickerFn()
	}
	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-ch:
				w.Poll()
			}
		}
	}()
}

// Stop halts polling and releases resources.
func (w *Watcher) Stop() {
	if w.done != nil {
		close(w.done)
		w.done = nil
	}
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

// Poll reads any new lines from the sidecar file since the last call.
// It is exported so tests can drive it synchronously.
func (w *Watcher) Poll() {
	f, err := os.Open(w.path)
	if err != nil {
		return // File may not exist yet; wait silently.
	}
	defer f.Close()

	// Determine current file size to detect new content.
	endPos, err := f.Seek(0, io.SeekEnd)
	if err != nil || endPos <= w.offset {
		return
	}

	// Seek to where we left off.
	if _, err = f.Seek(w.offset, io.SeekStart); err != nil {
		return
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		var ev entity.ClaudeEvent
		if err := json.Unmarshal([]byte(text), &ev); err != nil {
			log.Printf("eventwatch: malformed line: %v", err)
			continue
		}
		w.handler(ev)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("eventwatch: scan error: %v", err)
	}

	// Advance offset to the size captured at poll start. Any content written
	// after that snapshot is picked up on the next poll.
	w.offset = endPos
}
