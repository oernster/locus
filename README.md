# <img width="128" height="128" alt="appicon" src="https://github.com/user-attachments/assets/928c482c-7509-455b-b674-6cae4f9b337f" /> Locus

Locus is a native Windows productivity tool that merges two concerns into one surface:

- **Task board** — move work through a fixed four-stage workflow (Plan, Execute, Check, Done)
- **Focus intelligence** — automatic, OS-level tracking of which applications hold your attention, correlated against your task sessions

The top half of each column is yours to command. The bottom half shows what actually happened.

---

## Why Locus

Most task tools track intent. Most time trackers record raw clock time. Neither tells you whether the two matched.

Locus closes that gap:

- You say you were executing a task. Were you in your editor, or in Discord?
- You logged two hours on a review. How much was idle time?
- Your planning sessions — are you actually in research tools, or drifting?

Locus answers these without any manual input. It reads OS-level focus data from [focus-reader](https://github.com/oernster/focus-reader), which runs silently in the background and records which application holds foreground focus at every moment.

---

## Board model

Work is organised into **four fixed stages** with stable internal IDs:

`PLAN` · `EXECUTE` · `CHECK` · `DONE`

Stage labels are user-renameable per board. The number of stages and their order are fixed.

Each column is split horizontally:

| Half | Purpose | Editable |
|---|---|---|
| Top | Task cards (intent layer) | Yes |
| Bottom | Focus intelligence panel (reality layer) | No |

Bottom panel headers are fixed and describe the data, not the workflow:

| Column | Bottom header | What it surfaces |
|---|---|---|
| PLAN | EXPLORATION | App mix during planning sessions; drift signal |
| EXECUTE | DEEP WORK | Idle-corrected focus time; editor/terminal ratio |
| CHECK | ANALYSIS | Review tool fidelity |
| DONE | RETROSPECTIVE | Lifetime stats: total time, stage durations, completion date |

---

## Tasks

Tasks (internally: Commands) progress through a status model:

- Not Started
- In Progress
- Blocked
- Complete

Tasks are not plans. They are active units of execution.

---

## Sessions

Time is tracked at the **task level**.

Only one session can be active at a time. Starting a session requires selecting a task; the task's stage is pinned on the session row at start.

The DEEP WORK panel subtracts idle gaps (detected from focus-reader data) from raw session duration, giving honest active time.

---

## Focus intelligence

Locus reads `%APPDATA%\focus-reader\sessions.db` read-only at runtime. focus-reader must be installed and running for the bottom panels to show data.

If focus-reader is absent, the bottom panels display a prompt to install it. All task board functionality works regardless.

App names are resolved from Windows PE version info (`FileDescription` field), falling back to `ProductName`, then the executable stem. Results are cached in-memory. No hardcoded name mappings.

System processes (executables under `C:\Windows\`) are filtered out automatically.

---

## Stack

| Concern | Choice | Reason |
|---|---|---|
| Language | Go 1.21+ | Single binary, no runtime dependency |
| UI framework | Wails v2 + WebView2 | Native window, React frontend, full Win32 API access |
| Frontend | React 19 + TypeScript + Vite | Rich drag-and-drop board; CSS Modules |
| System tray | lxn/walk NotifyIcon | Consistent with focus-reader; runs in OS-thread-locked goroutine |
| Storage | modernc.org/sqlite | Pure Go, no CGO, no external DLL |
| OS API | golang.org/x/sys/windows | PE version info, Win32 direct access |
| Architecture | Clean Architecture (Domain / Application / Infrastructure / UI) | Enforced by AST boundary tests |

---

## Install

**Prerequisites:** Go 1.21+, Node.js 20+, Windows 10/11.

WebView2 runtime is required. It ships with Windows 11. For Windows 10, download from [Microsoft](https://developer.microsoft.com/en-us/microsoft-edge/webview2/).

### One-shot installer (recommended)

Run once from the project root. Builds, installs to `%LOCALAPPDATA%\locus\`, registers for auto-start on every login, and launches immediately.

```powershell
.\install.ps1
```

The Wails CLI is installed automatically if not already present. No administrator rights required.

### Uninstaller

Stops any running instance, removes the startup Run key, and deletes the install directory. Board data in `%APPDATA%\locus\` is kept by default so your tasks and history survive a reinstall.

```powershell
.\uninstall.ps1
```

To also delete all board data (tasks, sessions, outcomes, snapshots):

```powershell
.\uninstall.ps1 -PurgeData
```

### Manual / development build

```powershell
# Install Wails CLI (once)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install frontend dependencies
cd frontend && npm install && cd ..

# Development mode (hot-reload frontend + Go backend)
wails dev

# Production build (outputs build\bin\locus.exe)
wails build
```

The output is a single `locus.exe` with the frontend bundled inside.

For focus intelligence panels to populate, [focus-reader](https://github.com/oernster/focus-reader) must be installed and running alongside Locus.

---

## Snapshots

Snapshots serialize the full board state (tasks, outcomes, sessions) to a named JSON blob. They support:

- Save on demand
- Auto-save before destructive operations (reset, load)
- Structural deduplication (SHA-256 of task/outcome content, excluding timestamps)
- Rename and delete

Snapshot schema version 5. Older snapshots with CommandDeck stage IDs (DESIGN, BUILD, REVIEW, COMPLETE) are migrated automatically to Locus stage IDs (PLAN, EXECUTE, CHECK, DONE) on load.

---

## Storage

Locus is local-first. All data is stored in SQLite at:

```
%APPDATA%\locus\locus.db
```

No telemetry. No cloud sync. No accounts.

---

## Documentation

| Document | Contents |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Layer model, dependency rules, component map, execution flow, design decisions |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Prerequisites, local dev setup, project layout, build pipeline, packaging |
| [TESTING.md](TESTING.md) | Test categories, structural boundary tests, running the suite, coverage targets |

---

## License

GPL-3.0. See [LICENSE](LICENSE).
