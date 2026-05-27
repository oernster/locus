# Locus Architecture

## Overview

Locus is a Windows native application that combines task board management
(inspired by CommandDeck) with OS-level focus tracking (via focus-reader
integration). It uses Wails v2 to host a React 19 frontend inside a WebView2
window, with a system tray icon managed by lxn/walk.

## Layer Structure

```
Domain
  entity/        Pure Go structs and constants. No external dependencies.
Application
  dto/           Data transfer objects for the Wails IPC boundary.
  service/       Business logic. Depends on Domain only.
Infrastructure
  persistence/   SQLite repositories (modernc.org/sqlite, pure Go).
  focusreader/   Read-only access to focus-reader sessions.db.
  wininfo/       PE version-info lookup via Win32 API.
  startup/       Windows Run key registration.
  tray/          lxn/walk system tray (background goroutine).
UI
  app.go         Wails App struct: all bound methods.
  main.go        Entry point: DB init, dependency wiring, tray + Wails launch.
  frontend/      React 19 + TypeScript (Vite, CSS Modules).
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
  `-- wails.Run() (blocks on main goroutine)
```

When the user clicks "Open" in the tray: `runtime.WindowShow(ctx)`.
When the user clicks "Exit": `runtime.Quit(ctx)`.

## Data Storage

- Locus DB: `%APPDATA%\locus\locus.db` (read-write, owned by Locus).
- Focus-reader DB: `%APPDATA%\focus-reader\sessions.db` (read-only, owned by focus-reader).

## Focus Integration

FocusService correlates locus session time windows with focus-reader sessions.
For each completed locus session in a stage, it queries focus-reader for
overlapping application usage, aggregates per-executable time, subtracts idle
gaps exceeding 5 minutes, and returns the result as FocusDataDTO.

## Snapshot Schema

Snapshots use version 5 (PLAN/EXECUTE/CHECK/DONE stage IDs). On load, the
SnapshotService migrates older stage IDs: DESIGN->PLAN, BUILD->EXECUTE,
REVIEW->CHECK, COMPLETE->DONE.
