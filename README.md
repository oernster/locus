# <img width="128" height="128" alt="appicon" src="https://github.com/user-attachments/assets/928c482c-7509-455b-b674-6cae4f9b337f" /> Locus

Locus is a native Windows productivity tool that merges two concerns into one surface:

- **Task board**: move work through a fixed four-stage workflow (Plan, Execute, Check, Done)
- **Focus intelligence**: automatic, OS-level tracking of which applications hold your attention, built directly into Locus with no external tools required

# <img width="1264" height="785" alt="locus" src="https://github.com/user-attachments/assets/f1c2ce33-b67c-4b69-b49d-c8f460f6da21" />

---

## Why Locus

Most task tools track intent. Most time trackers record raw clock time. Neither tells you whether the two matched.

Locus closes that gap:

- You say you were executing a task. Were you in your editor, or in Discord?
- You logged two hours on a review. How much was idle time?
- Your planning sessions: are you actually in research tools, or drifting?

Locus answers these without any manual input. It tracks OS-level foreground focus natively, recording which application holds your attention at all times while it runs.

---

## UI Layout

```
╔════════════════════════════════════════════════════════╗
║  🏷  Board Name              ▶ Start    ＋ Add         ║
╠════════════════════════════════════════════════════════╣
║  FOCUS HISTORY  ▲                                      ║
║  ┌──────────┬───────────┬───────────┬────────────┐     ║
║  │  Today   │ Yesterday │ This Week │ This Month │     ║
║  └──────────┴───────────┴───────────┴────────────┘     ║
║  ████████████████████░░░░  VS Code        2h 14m       ║
║  ████████░░░░░░░░░░░░░░░░  Terminal         58m        ║
║  ███░░░░░░░░░░░░░░░░░░░░░  Chrome           21m        ║
╠════════════════════════════════════════════════════════╣
║  Reset board                        Snapshots ▼        ║
║  Tip: drag the grip to reorder or move between stages  ║
╠══════════╦══════════╦══════════╦═══════════════════════╣
║  PLAN    ║ EXECUTE  ║  CHECK   ║  DONE                 ║
║  · Task  ║  · Task  ║  · Task  ║  · Task               ║
║  · Task  ║  · Task  ║          ║                       ║
╚══════════╩══════════╩══════════╩═══════════════════════╝
```

**Focus History** is expanded by default. Click the header to collapse it, then pick a time period. It shows up to 10 apps ranked by total focus time, with a scrollable list if more than 5 are visible. Today's view refreshes every 2 seconds.

---

## Board model

Work is organised into **four fixed stages** with stable internal IDs:

`PLAN` · `EXECUTE` · `CHECK` · `DONE`

Stage labels are user-renameable per board. The number of stages and their order are fixed.

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

---

## Claude Code Integration

Locus populates the board automatically from an active Claude Code session. Hook scripts and a skill file are installed and uninstalled automatically by `install.ps1` / `uninstall.ps1`; no manual hook setup needed. Restart Claude CLI after installation for hooks to take effect.

Each `Edit`, `Write`, `NotebookEdit`, or `Bash` tool call appears as a dynamic board card with an **amber left border**:

- `Edit` / `Write` / `NotebookEdit` start at **EXECUTE**.
- `Bash` with test keywords (`go test`, `pytest`, `jest`, etc.) starts at **CHECK**.
- `Bash` with other meaningful commands starts at **EXECUTE**.
- On completion (success), the item moves to **DONE**. On failure, it stays in place.

**Noise filtering:** trivial commands (`cd`, `ls`, `grep`, `cat`, `echo`, `find`, `sed`, `awk`, `wc`, `sort`, PowerShell read-only equivalents, etc.) are silently dropped. Compound commands such as `cd /path && go test ./...` strip the trivial prefix first, so the meaningful part still produces a board item.

**Title classification:**

| Pattern | Label example |
|---|---|
| `git log`, `git diff`, etc. | `Git: log` |
| `go test`, `pytest`, `jest`, etc. | `Test: run suite` |
| `go build`, `wails build`, `make`, etc. | `Build: compile` |
| `npm install`, `go get`, `pip install`, etc. | `Install: dependencies` |
| `wails dev`, `go run`, `.exe`, `.sh`, etc. | `Launch: app` |
| `cp`, `mv`, `rm`, `mkdir`, etc. | `Files: organise` |
| Other | First 64 chars of command |

`Edit` / `Write` / `NotebookEdit` items are labelled `Edit: filename`, `Write: filename`, etc.

**Session end:** board items are kept. A snapshot named `Session YYYY-MM-DD HH:MM` is saved automatically. Items remain visible until you reset or clear the board manually.

Dynamic items coexist with manual items and can be dragged, renamed, and deleted like any manual card.

### Claude Code skill (optional but recommended)

`install.ps1` installs a skill file to `~/.claude/skills/locus/SKILL.md`, giving Claude persistent context about Locus without you having to explain it each session.

Load it at the start of any Claude Code session:

```
/go locus
```

Or load it automatically every session by adding this to `~/.claude/CLAUDE.md`:

```markdown
## Locus Integration

Locus hooks are active. At session start, load: ~/.claude/skills/locus/SKILL.md
```

---

## Focus intelligence

Locus tracks foreground focus natively using the Windows `GetForegroundWindow` and `QueryFullProcessImageNameW` APIs. No external tool is required.

The tracker polls every 500ms and writes to a `focus_sessions` table in the same SQLite database as your tasks (`%APPDATA%\locus\locus.db`). Focus data is available immediately from first launch.

**Focus History** shows:

| Period | Range |
|---|---|
| Today | Midnight local time to now |
| Yesterday | Previous calendar day |
| This Week | Monday to now (ISO week) |
| This Month | 1st of month to now |

App names are resolved from Windows PE version info (`FileDescription` field), falling back to `ProductName`, then the executable stem. Results are cached in-memory. No hardcoded name mappings.

System processes (executables under `C:\Windows\`) are filtered out automatically.

Idle gaps exceeding 5 minutes are detected and subtracted from deep work time.

---

## Stack

| Concern | Choice | Reason |
|---|---|---|
| Language | Go 1.21+ | Single binary, no runtime dependency |
| UI framework | Wails v2 + WebView2 | Native window, React frontend, full Win32 API access |
| Frontend | React 19 + TypeScript + Vite | Rich drag-and-drop board; CSS Modules |
| System tray | lxn/walk NotifyIcon | Runs in OS-thread-locked goroutine |
| Storage | modernc.org/sqlite | Pure Go, no CGO, no external DLL |
| OS API | golang.org/x/sys/windows + syscall | PE version info, focus tracking, Win32 direct access |
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

To also delete all board data (tasks, sessions, outcomes, snapshots, focus history):

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

Tables: `commands`, `sessions`, `outcomes`, `board_state`, `snapshots`, `focus_sessions`.

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
