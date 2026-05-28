#Requires -Version 5.1
<#
.SYNOPSIS
    Removes Locus from Windows startup and optionally purges all data.

.DESCRIPTION
    Stops any running Locus process, removes the HKCU Run key that causes
    it to launch at login, deletes the install directory (%LOCALAPPDATA%\locus\),
    and removes the Claude Code hook entries from ~/.claude/settings.json.

    By default the task database (%APPDATA%\locus\locus.db) is kept so your
    board, sessions, outcomes, and snapshots survive a reinstall.
    Pass -PurgeData to delete it too.

    No administrator rights required.

.PARAMETER PurgeData
    Also removes %APPDATA%\locus\ including the task database and all
    snapshots. All board history will be permanently lost.

.PARAMETER KeepClaudeHooks
    Skip removing the Claude Code hook entries from settings.json and
    the skill file from ~/.claude/skills/locus/.

.EXAMPLE
    .\uninstall.ps1
    .\uninstall.ps1 -PurgeData
    .\uninstall.ps1 -KeepClaudeHooks
#>
param(
    [switch]$PurgeData,
    [switch]$KeepClaudeHooks
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$AppName    = 'locus'
$RunKeyPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$InstallDir = Join-Path $env:LOCALAPPDATA $AppName
$DataDir    = Join-Path $env:APPDATA      $AppName

function Write-Step([string]$msg) {
    Write-Host "  $msg" -ForegroundColor Cyan
}

function Write-Ok([string]$msg) {
    Write-Host "  OK  $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "  WARN  $msg" -ForegroundColor Yellow
}

function Remove-ClaudeIntegration {
    $claudeSettingsPath = Join-Path $env:USERPROFILE '.claude\settings.json'
    $claudeSkillDir     = Join-Path $env:USERPROFILE '.claude\skills\locus'

    # Remove skill directory
    if (Test-Path $claudeSkillDir) {
        Remove-Item -Recurse -Force $claudeSkillDir
        Write-Ok "Claude skill removed from $claudeSkillDir"
    } else {
        Write-Ok "Claude skill directory not present (already clean)."
    }

    # Remove hooks from settings.json
    if (-not (Test-Path $claudeSettingsPath)) {
        Write-Ok "Claude Code settings.json not found; nothing to clean."
        return
    }

    $raw      = Get-Content $claudeSettingsPath -Raw -Encoding UTF8
    $settings = $raw | ConvertFrom-Json

    if (-not $settings.PSObject.Properties['hooks']) {
        Write-Ok "No hooks in settings.json; nothing to clean."
        return
    }

    $changed    = $false
    $eventNames = @('SessionStart', 'PreToolUse', 'PostToolUse', 'Stop')

    foreach ($event in $eventNames) {
        if (-not $settings.hooks.PSObject.Properties[$event]) { continue }

        $filtered = @()
        foreach ($group in $settings.hooks.$event) {
            if (-not $group.PSObject.Properties['hooks']) {
                $filtered += $group
                continue
            }
            $keptHooks = @($group.hooks | Where-Object {
                -not ($_.PSObject.Properties['command'] -and $_.command -match 'locus-')
            })
            if ($keptHooks.Count -gt 0) {
                $group | Add-Member -MemberType NoteProperty -Name 'hooks' -Value $keptHooks -Force
                $filtered += $group
            }
            if ($keptHooks.Count -ne $group.hooks.Count) { $changed = $true }
        }

        $settings.hooks | Add-Member -MemberType NoteProperty -Name $event -Value $filtered -Force
    }

    if ($changed) {
        $settings | ConvertTo-Json -Depth 10 | Set-Content $claudeSettingsPath -Encoding UTF8
        Write-Ok "Claude Code hooks removed from $claudeSettingsPath"
    } else {
        Write-Ok "No Locus hooks found in settings.json (already clean)."
    }
}

Write-Host "`nUninstalling $AppName..." -ForegroundColor Yellow

# 1. Stop any running instance
Write-Step "Stopping any running instance..."
$procs = Get-Process -Name $AppName -ErrorAction SilentlyContinue
if ($procs) {
    $procs | Stop-Process -Force
    Start-Sleep -Milliseconds 600
    Write-Ok "Process stopped."
} else {
    Write-Ok "No running instance found."
}

# 2. Remove Run key
Write-Step "Removing startup Run key..."
$existing = try {
    (Get-ItemProperty -Path $RunKeyPath -Name $AppName -ErrorAction Stop).$AppName
} catch { $null }
if ($existing) {
    Remove-ItemProperty -Path $RunKeyPath -Name $AppName -ErrorAction SilentlyContinue
    Write-Ok "Run key removed."
} else {
    Write-Ok "Run key not present (already clean)."
}

# 3. Remove install directory
Write-Step "Removing install directory ($InstallDir)..."
if (Test-Path $InstallDir) {
    Remove-Item -Recurse -Force $InstallDir
    Write-Ok "Install directory removed."
} else {
    Write-Ok "Install directory not found (already clean)."
}

# 4. Remove Claude CLI integration
if (-not $KeepClaudeHooks) {
    Write-Step "Removing Claude Code integration..."
    Remove-ClaudeIntegration
} else {
    Write-Ok "Claude Code integration kept (-KeepClaudeHooks)."
}

# 5. Optionally purge data
if ($PurgeData) {
    Write-Step "Removing data directory ($DataDir)..."
    if (Test-Path $DataDir) {
        Remove-Item -Recurse -Force $DataDir
        Write-Ok "Data directory removed. All board history deleted."
    } else {
        Write-Ok "Data directory not found (already clean)."
    }
} else {
    if (Test-Path $DataDir) {
        Write-Warn "Board data kept at $DataDir -- run with -PurgeData to delete it."
    }
}

Write-Host "`n$AppName has been removed.`n" -ForegroundColor Green
if (-not $PurgeData -and (Test-Path $DataDir)) {
    Write-Host "  Board data preserved at $DataDir`n" -ForegroundColor Gray
    Write-Host "  Run .\uninstall.ps1 -PurgeData to also delete it.`n" -ForegroundColor Gray
}
