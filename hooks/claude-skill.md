---
name: locus
description: "Locus integration context. Locus is a native Windows productivity app that mirrors Claude Code tool calls onto a live task board in real time. Load this skill when working in any project that has the Locus hooks installed. Tells Claude what appears on the board, what is filtered, and how the session lifecycle works. Trigger: 'load locus', '/go locus', or whenever Locus is mentioned in the session."
type: integration
platform: windows
version: "1.0"
---

# Locus — Claude Code Integration Context

Locus is a native Windows desktop app (system tray, always-on) that combines a four-stage task board with OS-level focus tracking. When the Locus hooks are installed in your Claude Code settings, tool calls made during this session appear on the board as live cards.

## What appears on the board

Only these tools produce board items:

| Tool | Condition | Board card |
|---|---|---|
| `Edit` | any file | `Edit: <filename>` at EXECUTE |
| `Write` | any file | `Write: <filename>` at EXECUTE |
| `NotebookEdit` | any notebook | `NotebookEdit: <filename>` at EXECUTE |
| `Bash` | non-trivial command | classified label (see below) at EXECUTE or CHECK |

All other tools (`Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`, `Agent`, etc.) produce no board item.

## Bash classification

Bash commands are classified into readable labels before appearing on the board:

| Command pattern | Label | Stage |
|---|---|---|
| `git <sub>` | `Git: <sub>` | EXECUTE |
| `go test`, `pytest`, `jest`, `mocha`, etc. | `Test: run suite` | CHECK |
| `go build`, `wails build`, `make`, etc. | `Build: compile` | EXECUTE |
| `npm install`, `go get`, `pip install`, etc. | `Install: dependencies` | EXECUTE |
| `wails dev`, `go run`, `.exe`, `.sh` | `Launch: app` | EXECUTE |
| `cp`, `mv`, `rm`, `mkdir`, etc. | `Files: organise` | EXECUTE |
| other | first 64 chars of command | EXECUTE |

## Noise filtering

Trivial Bash commands are silently dropped and never appear on the board:

- Navigation: `cd`, `pwd`, `ls`, `dir`
- Output: `echo`, `printf`
- Search: `grep`, `rg`, `find`, `locate`
- Read: `cat`, `head`, `tail`, `less`, `more`
- Transform: `sed`, `awk`, `wc`, `sort`, `uniq`, `cut`, `tr`
- Info: `which`, `where`, `type`, `date`, `env`
- PowerShell read-only: `Get-ChildItem`, `Select-String`, `Get-Content`, `Write-Output`, `Test-Path`, `Get-Location`, `Measure-Object`

Compound commands like `cd /path && go test ./...` strip the leading trivial segment before checking: `go test ./...` is non-trivial and produces `Test: run suite`.

## Item lifecycle

1. `tool_use` event: card created at inferred stage.
2. `tool_result` success: card moves directly to DONE with status Complete.
3. `tool_result` failure: card stays at its current stage.
4. Session end: cards remain on the board. A snapshot named `"Session YYYY-MM-DD HH:MM"` is saved automatically.

Cards have an amber left border to distinguish them from manual cards. They can be dragged, renamed, and deleted like any manual card.

## Board model

Four fixed stages (PLAN, EXECUTE, CHECK, DONE). Each card has a status (Not Started, In Progress, Blocked, Complete). Dynamic Claude-sourced cards arrive at EXECUTE or CHECK and move to DONE on success.

## Events file

Hook output: `%LOCALAPPDATA%\Locus\events.jsonl`
Database: `%APPDATA%\locus\locus.db`
Locus polls events every 500ms and emits a board-update signal to the frontend on each mutation.

## Implications for this session

- File edits, writes, and meaningful shell commands are visible on the Locus board in real time.
- Read-only research (Grep, Read, Glob, WebSearch) is invisible to Locus — only actions that change state appear.
- Locus does not influence Claude's behavior — it is an observer, not a controller.
- If the user says "check the board" or "what's on the board", they mean the Locus task board, not an internal task list.
