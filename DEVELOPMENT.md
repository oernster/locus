# Locus Development Guide

## Prerequisites

- Go 1.21 or later
- Node.js 20 or later (for the React frontend)
- Wails CLI v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 runtime (ships with Windows 11; downloadable for Windows 10)

## Getting Started

```powershell
cd C:\path\to\locus

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode (hot-reload frontend + Go backend)
wails dev

# Build production binary
wails build
```

Output: `build\bin\locus.exe` — single binary with frontend embedded.

## Project Layout

```
locus/
  main.go                   Entry point: DB init, tracker start, wiring, tray + Wails launch
  app.go                    Wails IPC bound methods
  icon_windows.go           Taskbar/window icon setup (Windows only)
  wails.json                Wails configuration
  go.mod / go.sum
  install.ps1               One-shot installer (build + register Run key + launch)
  uninstall.ps1             Uninstaller
  Makefile
  internal/
    domain/
      entity/               Stage, Command, Session, Outcome, BoardState, Snapshot
      repository/           Repository interfaces (5 interfaces)
    application/
      dto/                  Command, Session, Outcome, Board, Snapshot, Focus DTOs
      service/              Command, Session, Outcome, Board, Snapshot, Focus services
    infrastructure/
      persistence/          SQLite: database.go (schema), 5 repository implementations
      focustracker/         tracker_windows.go — foreground window polling
      focusreader/          sqlite_focus_reader.go — focus_sessions aggregation
      wininfo/              app_info.go — PE version info lookup
      startup/              registry_startup.go — HKCU Run key
      tray/                 tray.go — lxn/walk NotifyIcon
  tests/
    structural/             boundary_test.go — Clean Architecture layer enforcement
  frontend/
    src/
      features/
        commands/           Board.tsx, CommandDrawer, CreateCommandModal, constants
        focus/              FocusHistory.tsx — collapsible focus panel with period picker
      components/           DestructiveGuardModal, ConfirmDangerModal
      types/                locus.ts — TypeScript type definitions
    index.html
    vite.config.ts
    tsconfig*.json
  build/
    appicon.png
    windows/                info.json, icon.ico, wails.exe.manifest
```

## Focus Tracking

The focus tracker (`internal/infrastructure/focustracker/tracker_windows.go`) starts automatically in `main()` before Wails launches. It polls `GetForegroundWindow` every 500ms and writes to the `focus_sessions` table in locus.db.

No external tools or databases are required. Focus data is available from first launch.

## Running Tests

```powershell
# All Go tests (requires Windows for platform tests)
go test ./...

# Structural boundary tests only
go test ./tests/structural/...
```

## Database Location

The SQLite database is created automatically at first run:

```
%APPDATA%\locus\locus.db
```

Schema is applied via `internal/infrastructure/persistence/database.go`. Tables: `commands`, `sessions`, `outcomes`, `board_state`, `snapshots`, `focus_sessions`.

## Startup Registration

Done automatically by `install.ps1`. Manual equivalent:

```powershell
Set-ItemProperty `
  -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' `
  -Name 'locus' `
  -Value '"C:\Users\<you>\AppData\Local\locus\locus.exe"'
```

## IPC Boundary

All Go methods exposed to the frontend are on the `App` struct in `app.go`. Wails generates TypeScript bindings into `frontend/wailsjs/go/main/App.ts` at build time. After adding or renaming a method, run `wails build` (or `wails dev`) to regenerate bindings.

Key IPC methods:

| Method | Purpose |
|---|---|
| `GetFocusData(stageId)` | Focus data correlated with locus sessions for a stage |
| `GetFocusDataForTimeRange(start, end)` | Focus data for arbitrary Unix second range (Focus History) |
| `StartSession(commandId)` | Begin a work session |
| `StopSession()` | End the active session |
| `ListCommands(stageId)` | Fetch all tasks, optionally filtered by stage |
