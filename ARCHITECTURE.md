# Locus Architecture

## Overview

Locus is a Windows native application that combines task board management
with OS-level focus tracking built directly into the process. It uses Wails v2
to host a React 19 frontend inside a WebView2 window, with a system tray icon
managed by lxn/walk.

## Layer Structure

```
Domain
  entity/          Pure Go structs and constants. No external dependencies.
Application
  dto/             Data transfer objects for the Wails IPC boundary.
  service/         Business logic. Depends on Domain only.
Infrastructure
  persistence/     SQLite repositories (modernc.org/sqlite, pure Go).
                   Owns the focus_sessions table alongside tasks/sessions.
  focustracker/    Built-in foreground-window poller (Windows only).
                   Writes to focus_sessions in locus.db every 500ms.
  focusreader/     Reads and aggregates focus_sessions from locus.db.
  eventwatch/      JSONL sidecar file poller. Reads %LOCALAPPDATA%\Locus\events.jsonl
                   every 500ms and dispatches ClaudeEvents to ClaudeSessionService.
  wininfo/         PE version-info lookup via Win32 API.
  startup/         Windows Run key registration.
  tray/            lxn/walk system tray (background goroutine).
UI
  app.go           Wails App struct: all bound methods.
  main.go          Entry point: DB init, dependency wiring, tray + Wails launch.
  frontend/        React 19 + TypeScript (Vite, CSS Modules).
```

## Dependency Rules

- Domain: imports stdlib only.
- Application: imports Domain and stdlib only.
- Infrastructure: imports Domain, Application, and stdlib. Never imported by Domain or Application.
- UI: imports Application services and Wails runtime. Never imports Infrastructure directly.

These rules are enforced by `tests/structural/boundary_test.go`.

## Wails + Walk Coordination

Wails requires the main goroutine. Walk requires an OS-thread-locked goroutine.

```
main()
  |-- Start walk tray goroutine (runtime.LockOSThread inside)
  |     Creates hidden MainWindow + NotifyIcon
  |     Closes ready channel when setup complete
  |     Blocks in mw.Run()
  |
  |-- <-ready  (wait for tray to be set up)
  |
  |-- Start focus tracker goroutine
  |     Polls GetForegroundWindow every 500ms
  |     Writes to focus_sessions table in locus.db
  |
  `-- wails.Run() (blocks on main goroutine)
```

When the user clicks "Open" in the tray: `runtime.WindowShow(ctx)`.
When the user clicks "Exit": `runtime.Quit(ctx)`.

## Data Storage

All data is stored in a single SQLite database:

```
%APPDATA%\locus\locus.db
```

Tables:

| Table | Owner | Purpose |
|---|---|---|
| commands | Locus | Task definitions (manual and dynamic Claude-sourced) |
| sessions | Locus | Work session time ranges per task/stage |
| outcomes | Locus | Notes attached to tasks |
| board_state | Locus | Board name and stage label overrides |
| snapshots | Locus | Serialised board snapshots (JSON) |
| focus_sessions | FocusTracker | Foreground app exe path + start/end Unix seconds |

There is no external database dependency. All read and write operations use the same `*sql.DB` instance opened in `main()`.

### Dynamic item columns (commands table)

| Column | Type | Purpose |
|---|---|---|
| source | TEXT DEFAULT 'manual' | `"manual"` or `"claude"` |
| session_id | TEXT DEFAULT '' | Claude session UUID; empty for manual items |
| archived_at | INTEGER | Unix timestamp; NULL = active. Reserved for future use; not set automatically. |

`List()` always filters `WHERE archived_at IS NULL` so archived items are invisible to the board.

## Focus Tracking

### Tracker (Infrastructure/focustracker)

`tracker_windows.go` runs a background goroutine that:

1. Calls `GetForegroundWindow()` every 500ms.
2. Resolves the window's process exe path via `OpenProcess` + `QueryFullProcessImageNameW`.
3. On foreground change: closes the previous `focus_sessions` row (`ended_at = now`), inserts a new row for the new exe.
4. On startup: closes any stale rows with `ended_at IS NULL` left by a prior crash.

### Reader (Infrastructure/focusreader)

`SQLiteFocusReader` queries `focus_sessions` and aggregates per-exe duration over supplied time windows. It:

- Clamps sessions to the requested window boundaries.
- Filters out `C:\Windows\` system processes.
- Detects idle gaps exceeding 5 minutes and subtracts them from deep work time.
- Returns up to 10 apps ranked by total seconds.

### Service (Application/service)

`FocusService` exposes two query paths:

- `GetFocusDataForStage(stageId)`: correlates focus data with locus session time windows for a stage. Falls back to a 2-hour rolling window if no sessions exist yet.
- `GetFocusDataForTimeRange(startUnix, endUnix)`: returns aggregated focus data for an arbitrary time range with no stage correlation. Used by the Focus History UI.

### Frontend (FocusHistory)

The collapsible Focus History panel sits above the board columns. The frontend computes calendar boundaries (Today / Yesterday / This Week / This Month) in local time and passes Unix second timestamps to `GetFocusDataForTimeRange`. Today's view polls every 2 seconds for live updates.

## Claude Code Integration

### Overview

When Claude Code is running with the Locus hooks installed, tool calls are
reflected live on the board as dynamic items with an amber left border.

```
Claude Code hook (PreToolUse / PostToolUse / Stop / SessionStart)
  |-- runs locus-*.js
  |-- appends JSON line to %LOCALAPPDATA%\Locus\events.jsonl

eventwatch.Watcher (goroutine, 500ms poll)
  |-- on Start(): seeks to EOF so app restart does not replay prior history
  |-- detects new lines appended since last poll
  |-- dispatches ClaudeEvent to ClaudeSessionService.HandleEvent

ClaudeSessionService
  |-- tool_use (item tool, non-trivial): creates Command{source="claude", session_id=...}
  |-- tool_use (trivial Bash / non-item tool): silently dropped
  |-- tool_result (success): sets item to StageDone + StatusComplete
  |-- tool_result (failure): item stays at current stage
  |-- session_end: cleans pending stack, takes auto-snapshot, notifies board
  |-- each mutation: calls notifyFn -> signals boardNotify channel

App.startup goroutine
  |-- drains boardNotify channel
  |-- calls runtime.EventsEmit(ctx, "locus:board-updated")

Frontend Board.tsx
  |-- EventsOn("locus:board-updated") triggers refresh()
  |-- dynamic items rendered with styles.cardDynamic (amber left border)
```

### Noise filtering

`isTrivialBash(cmd)` drops Bash commands that produce no conceptual board item:
read-only operations (`cd`, `ls`, `grep`, `cat`, `echo`, `find`, `sed`, `awk`,
`wc`, `sort`, PowerShell read-only equivalents, etc.).

`meaningfulSegment(cmd)` strips leading trivial segments from compound commands
before the triviality check. Example: `cd /path && go test ./...` strips `cd /path &&`
and tests `go test ./...`, which is non-trivial.

### Title classification

`formatTitle(tool, target)` dispatches to `fileTitle` or `bashTitle`:

- `fileTitle`: `Edit / Write / NotebookEdit` -> `"Edit: filename"` (basename, max 50 chars).
- `bashTitle`: classifies Bash commands into readable labels:

| Pattern | Label |
|---|---|
| `git <sub>` | `Git: <sub>` |
| test keywords | `Test: run suite` |
| build keywords | `Build: compile` |
| install keywords | `Install: dependencies` |
| launch keywords | `Launch: app` |
| file-op keywords | `Files: organise` |
| fallback | first 64 chars of command |

### Stage inference

| Tool | Bash keyword match | Stage |
|---|---|---|
| Edit, Write, NotebookEdit | (n/a) | EXECUTE |
| Bash | go test, npm test, pytest, jest, ... | CHECK |
| Bash | (other non-trivial) | EXECUTE |
| any other tool | (n/a) | PLAN (not shown on board; filtered) |

### Session lifecycle

1. `session_start`: hook fires, event written to sidecar (informational only).
2. Per tool call: `tool_use` creates board item; `tool_result` success moves item to DONE.
3. `session_end`: hook fires; pending stack cleaned; auto-snapshot saved as `"Session YYYY-MM-DD HH:MM"`.
4. Board items from the session **remain visible** after session end. User clears manually.

### Hook installation

See `hooks/README.md` for Claude Code settings.json configuration.

## Snapshot Schema

Snapshots use version 5 (PLAN/EXECUTE/CHECK/DONE stage IDs). On load, the
SnapshotService migrates older stage IDs: DESIGN->PLAN, BUILD->EXECUTE,
REVIEW->CHECK, COMPLETE->DONE.
