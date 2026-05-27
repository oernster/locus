//go:build windows

package focustracker

import (
	"database/sql"
	"log"
	"syscall"
	"time"
	"unsafe"
)

const (
	// processQueryLimitedInformation is the minimum access right needed to call
	// QueryFullProcessImageName.
	processQueryLimitedInformation = uintptr(0x1000)
	// pollInterval is how often the foreground window is sampled.
	pollInterval = 500 * time.Millisecond
	// exePathBufSize is the buffer length for QueryFullProcessImageNameW.
	exePathBufSize = 260
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId  = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

// Tracker polls the foreground window and records focus sessions in locus.db.
type Tracker struct {
	db   *sql.DB
	stop chan struct{}
}

// New creates a Tracker backed by the supplied locus DB.
func New(db *sql.DB) *Tracker {
	return &Tracker{db: db, stop: make(chan struct{})}
}

// Start closes any stale open sessions from a prior crash, then launches
// the polling loop in a background goroutine.
func (t *Tracker) Start() {
	t.closeStale()
	go t.run()
}

// Stop signals the polling loop to exit cleanly, ending the current session.
func (t *Tracker) Stop() {
	close(t.stop)
}

// closeStale ends focus_sessions left open by a previous crash.
func (t *Tracker) closeStale() {
	now := time.Now().Unix()
	if _, err := t.db.Exec(
		`UPDATE focus_sessions SET ended_at = ? WHERE ended_at IS NULL`, now); err != nil {
		log.Printf("focus tracker: close stale: %v", err)
	}
}

func (t *Tracker) run() {
	var currentExe string
	var currentID int64

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stop:
			if currentID != 0 {
				t.endSession(currentID)
			}
			return
		case <-ticker.C:
			exe := foregroundExe()
			if exe == "" || exe == currentExe {
				continue
			}
			if currentID != 0 {
				t.endSession(currentID)
			}
			currentExe = exe
			currentID = t.startSession(exe)
		}
	}
}

func (t *Tracker) startSession(exePath string) int64 {
	res, err := t.db.Exec(
		`INSERT INTO focus_sessions (exe_path, started_at) VALUES (?, ?)`,
		exePath, time.Now().Unix())
	if err != nil {
		log.Printf("focus tracker: insert: %v", err)
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}

func (t *Tracker) endSession(id int64) {
	if _, err := t.db.Exec(
		`UPDATE focus_sessions SET ended_at = ? WHERE id = ?`,
		time.Now().Unix(), id); err != nil {
		log.Printf("focus tracker: update: %v", err)
	}
}

// foregroundExe returns the full exe path of the currently focused window,
// or an empty string if it cannot be determined.
func foregroundExe() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	handle, _, _ := procOpenProcess.Call(
		processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	buf := make([]uint16, exePathBufSize)
	size := uint32(len(buf))
	procQueryFullProcessImageName.Call(
		handle, 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}
