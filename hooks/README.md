# Locus Claude Code Hooks

These hooks connect Claude Code to Locus so that tool calls made during a Claude
session automatically populate the Locus board with dynamic items.

## How it works

1. Each hook writes one JSON line to `%LOCALAPPDATA%\Locus\events.jsonl`.
2. Locus polls that file every 500ms and processes new events.
3. Item-creating tools (`Edit`, `Write`, `NotebookEdit`, `Bash`) appear as board cards.
4. Stage is inferred from tool type: Edit/Write/NotebookEdit = EXECUTE, Bash tests = CHECK, Bash other = EXECUTE.
5. Each item moves directly to DONE when its tool completes successfully. Failures leave the item in place.
6. Trivial Bash commands (`cd`, `ls`, `grep`, `cat`, `echo`, `find`, `sed`, `awk`, `wc`, `sort`, PowerShell read-only equivalents, etc.) are silently dropped.
7. Compound Bash commands with trivial prefixes (`cd /path && go test ./...`) strip the prefix before the triviality check.
8. When the Claude session ends, board items **remain on the board**. A snapshot named `"Session YYYY-MM-DD HH:MM"` is saved automatically.

**Title classification:** Bash commands are labelled by pattern:
`Git: <sub>` / `Test: run suite` / `Build: compile` / `Install: dependencies` / `Launch: app` / `Files: organise` / first 64 chars as fallback.

Dynamic items have an amber left border to distinguish them from manual cards.
They can be dragged, renamed, and deleted just like manual items.

## Installation

Add the following to your Claude Code settings file.
On Windows: `%APPDATA%\Claude\claude_desktop_config.json`
On Mac/Linux: `~/.claude/settings.json`

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node C:\\path\\to\\locus\\hooks\\locus-session-start.js"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node C:\\path\\to\\locus\\hooks\\locus-pre-tool.js"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node C:\\path\\to\\locus\\hooks\\locus-post-tool.js"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node C:\\path\\to\\locus\\hooks\\locus-stop.js"
          }
        ]
      }
    ]
  }
}
```

Replace `C:\\path\\to\\locus` with the actual path to your Locus repository.

## Claude Code skill

`claude-skill.md` (this directory) is a Claude Code skill file that gives Claude
persistent context about Locus - what appears on the board, what is filtered, and
how the session lifecycle works.

Install it once:

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.claude\skills\locus"
Copy-Item ".\hooks\claude-skill.md" "$env:USERPROFILE\.claude\skills\locus\SKILL.md"
```

Then load it at the start of any Claude Code session with `/go locus`.

## Requirements

- Node.js installed and available on PATH.
- Locus running in the system tray.

## Sidecar file

Events are appended to `%LOCALAPPDATA%\Locus\events.jsonl`.
This file grows over time. Locus does not prune it automatically; delete it
manually to reset. Each line is one JSON object:

```
{"type":"session_start","session_id":"abc","ts":1234567890}
{"type":"tool_use","session_id":"abc","tool":"Edit","target":"auth.go","ts":1234567891}
{"type":"tool_result","session_id":"abc","tool":"Edit","target":"auth.go","success":true,"ts":1234567892}
{"type":"session_end","session_id":"abc","ts":1234567893}
```
