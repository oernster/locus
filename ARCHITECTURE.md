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
| commands | Locus | Task definitions |
| sessions | Locus | Work session time ranges per task/stage |
| outcomes | Locus | Notes attached to tasks |
| board_state | Locus | Board name and stage label overrides |
| snapshots | Locus | Serialised board snapshots (JSON) |
| focus_sessions | FocusTracker | Foreground app exe path + start/end Unix seconds |

There is no external database dependency. All read and write operations use the same `*sql.DB` instance opened in `main()`.

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

## Snapshot Schema

Snapshots use version 5 (PLAN/EXECUTE/CHECK/DONE stage IDs). On load, the
SnapshotService migrates older stage IDs: DESIGN->PLAN, BUILD->EXECUTE,
REVIEW->CHECK, COMPLETE->DONE.
