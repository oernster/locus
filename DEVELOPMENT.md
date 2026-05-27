# Locus Development Guide

## Prerequisites

- Go 1.21 or later
- Node.js 20 or later (for the React frontend)
- Wails CLI v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 runtime (ships with Windows 11; downloadable for Windows 10)

## Getting Started

```powershell
cd C:\Users\Oliver\Development\locus

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode (hot-reload frontend + Go backend)
wails dev

# Build production binary
wails build
```

## Project Layout

```
locus/
  main.go              Entry point
  app.go               Wails-bound methods
  wails.json           Wails configuration
  go.mod
  go.sum
  internal/
    domain/            Pure domain layer
    application/       Services and DTOs
    infrastructure/    SQLite, tray, wininfo, startup
  tests/structural/    Layer boundary tests
  frontend/
    src/
      features/
        commands/      Board, CommandDrawer, CreateCommandModal
        focus/         FocusPanel
      components/      Modal dialogs
      types/           TypeScript types
```

## Running Tests

```powershell
# All Go tests (requires Windows for platform tests)
go test ./...

# Structural boundary tests only
go test ./tests/structural/...
```

## Database Location

The SQLite database is created automatically at first run:
`%APPDATA%\locus\locus.db`

## Startup Registration

To register Locus to run at login:
```powershell
# Done automatically by the installer. Manual equivalent:
$exe = (Get-Command .\locus.exe).Source
Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'locus' -Value $exe
```
