#Requires -Version 5.1
<#
.SYNOPSIS
    Installs Locus to run automatically at every Windows login.

.DESCRIPTION
    Builds locus.exe via Wails, copies it to %LOCALAPPDATA%\locus\,
    registers it in HKCU\Software\Microsoft\Windows\CurrentVersion\Run so it
    survives reboots, and starts it immediately.

    Also installs Claude Code hooks and skill file if Claude CLI is present.
    Restart Claude CLI after installation for the Locus integration to activate.

    Run once. Never think about it again.

    No administrator rights required.

.PARAMETER SkipClaudeHooks
    Skip Claude CLI hook installation even if Claude CLI is detected.

.EXAMPLE
    .\install.ps1
    .\install.ps1 -SkipClaudeHooks
#>
param(
    [switch]$SkipClaudeHooks
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$AppName    = 'locus'
$RunKeyPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$InstallDir = Join-Path $env:LOCALAPPDATA $AppName
$ExeDst     = Join-Path $InstallDir "$AppName.exe"
$ExeSrc     = Join-Path $PSScriptRoot "$AppName.exe"

function Write-Step([string]$msg) {
    Write-Host "  $msg" -ForegroundColor Cyan
}

function Write-Ok([string]$msg) {
    Write-Host "  OK  $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "  WARN  $msg" -ForegroundColor Yellow
}

function Write-Fail([string]$msg) {
    Write-Host "FAIL  $msg" -ForegroundColor Red
    exit 1
}

function Install-ClaudeIntegration {
    param([string]$HooksInstallDir)

    $claudeSettingsPath = Join-Path $env:USERPROFILE '.claude\settings.json'
    $claudeSkillDir     = Join-Path $env:USERPROFILE '.claude\skills\locus'

    # Install skill file
    $skillSrc = Join-Path $PSScriptRoot 'hooks\claude-skill.md'
    if (Test-Path $skillSrc) {
        New-Item -ItemType Directory -Force -Path $claudeSkillDir | Out-Null
        Copy-Item -Force $skillSrc (Join-Path $claudeSkillDir 'SKILL.md')
        Write-Ok "Claude skill installed to $claudeSkillDir\SKILL.md"
    }

    # Detect settings.json
    if (-not (Test-Path $claudeSettingsPath)) {
        Write-Warn "Claude Code settings.json not found at $claudeSettingsPath"
        Write-Warn "Start Claude CLI once to create it, then re-run install.ps1 to register hooks."
        return
    }

    # Detect node
    $nodeExe = (Get-Command node -ErrorAction SilentlyContinue)?.Source
    if (-not $nodeExe) {
        Write-Warn "node.exe not found in PATH. Hooks not registered in settings.json."
        Write-Warn "Install Node.js then re-run install.ps1."
        return
    }

    # Hook event -> script filename
    $hookMap = [ordered]@{
        SessionStart = 'locus-session-start.js'
        PreToolUse   = 'locus-pre-tool.js'
        PostToolUse  = 'locus-post-tool.js'
        Stop         = 'locus-stop.js'
    }

    # Load settings
    $raw      = Get-Content $claudeSettingsPath -Raw -Encoding UTF8
    $settings = $raw | ConvertFrom-Json

    if (-not $settings.PSObject.Properties['hooks']) {
        $settings | Add-Member -MemberType NoteProperty -Name 'hooks' -Value ([PSCustomObject]@{})
    }

    $changed = $false

    foreach ($event in $hookMap.Keys) {
        $scriptPath = Join-Path $HooksInstallDir $hookMap[$event]
        $cmd        = "`"$nodeExe`" `"$scriptPath`""

        if (-not $settings.hooks.PSObject.Properties[$event]) {
            $settings.hooks | Add-Member -MemberType NoteProperty -Name $event -Value @() -Force
        }

        # Check if locus hook already registered for this event
        $alreadyPresent = $false
        foreach ($group in $settings.hooks.$event) {
            if ($group.PSObject.Properties['hooks']) {
                foreach ($h in $group.hooks) {
                    if ($h.PSObject.Properties['command'] -and $h.command -match 'locus-') {
                        $alreadyPresent = $true
                        break
                    }
                }
            }
            if ($alreadyPresent) { break }
        }

        if (-not $alreadyPresent) {
            $newEntry = [PSCustomObject]@{
                hooks = @(
                    [PSCustomObject]@{ type = 'command'; command = $cmd; timeout = 5 }
                )
            }
            $currentArr = @($settings.hooks.$event)
            $currentArr += $newEntry
            $settings.hooks | Add-Member -MemberType NoteProperty -Name $event -Value $currentArr -Force
            $changed = $true
        }
    }

    if ($changed) {
        $settings | ConvertTo-Json -Depth 10 | Set-Content $claudeSettingsPath -Encoding UTF8
        Write-Ok "Claude Code hooks registered in $claudeSettingsPath"
        Write-Host "  NOTE  Restart Claude CLI to activate Locus integration." -ForegroundColor Yellow
    } else {
        Write-Ok "Claude Code hooks already registered (no changes made)."
    }
}

Write-Host "`nInstalling $AppName..." -ForegroundColor Yellow

# 1. Check prerequisites
Write-Step "Checking prerequisites..."

$goExe = (Get-Command go -ErrorAction SilentlyContinue)?.Source
if (-not $goExe) { Write-Fail "Go not found in PATH. Install from https://go.dev/dl/" }

$wailsExe = (Get-Command wails -ErrorAction SilentlyContinue)?.Source
if (-not $wailsExe) {
    Write-Step "Wails CLI not found. Installing..."
    & go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if ($LASTEXITCODE -ne 0) { Write-Fail "Failed to install Wails CLI." }
    Write-Ok "Wails CLI installed."
}

$nodeExe = (Get-Command node -ErrorAction SilentlyContinue)?.Source
if (-not $nodeExe) { Write-Fail "Node.js not found in PATH. Install from https://nodejs.org/" }

Write-Ok "Prerequisites satisfied."

# 2. Install frontend dependencies
Write-Step "Installing frontend dependencies..."
Push-Location (Join-Path $PSScriptRoot 'frontend')
try {
    & npm install --silent
    if ($LASTEXITCODE -ne 0) { Write-Fail "npm install failed (exit $LASTEXITCODE)" }
} finally {
    Pop-Location
}
Write-Ok "Frontend dependencies ready."

# 3. Build with Wails
Write-Step "Building $AppName.exe (this may take a moment)..."
Push-Location $PSScriptRoot
try {
    & wails build
    if ($LASTEXITCODE -ne 0) { Write-Fail "wails build failed (exit $LASTEXITCODE)" }
} finally {
    Pop-Location
}

# Wails outputs to build\bin\ by default
$WailsBin = Join-Path $PSScriptRoot "build\bin\$AppName.exe"
if (Test-Path $WailsBin) {
    Copy-Item -Force $WailsBin $ExeSrc
}

if (-not (Test-Path $ExeSrc)) {
    Write-Fail "Expected binary not found after build. Checked: $ExeSrc and $WailsBin"
}
Write-Ok "Build complete."

# 4. Stop any existing instance before overwriting
Write-Step "Stopping any existing instance..."
Get-Process -Name $AppName -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 500

# 5. Copy to install directory
Write-Step "Installing to $InstallDir..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item -Force $ExeSrc $ExeDst
Write-Ok "Copied to $ExeDst"

# 6. Copy hooks to install directory
Write-Step "Copying Claude Code hooks..."
$HooksInstallDir = Join-Path $InstallDir 'hooks'
New-Item -ItemType Directory -Force -Path $HooksInstallDir | Out-Null
Copy-Item -Force (Join-Path $PSScriptRoot 'hooks\*.js') $HooksInstallDir
Write-Ok "Hooks copied to $HooksInstallDir"

# 7. Register in HKCU Run (survives reboots, runs in user session, no admin required)
Write-Step "Registering in Run key..."
Set-ItemProperty -Path $RunKeyPath -Name $AppName -Value "`"$ExeDst`""
$registered = (Get-ItemProperty -Path $RunKeyPath -Name $AppName -ErrorAction SilentlyContinue).$AppName
if (-not $registered) { Write-Fail "Failed to write Run key." }
Write-Ok "Registered: $registered"

# 8. Start immediately
Write-Step "Starting $AppName..."
Start-Process -FilePath $ExeDst -WindowStyle Normal
Start-Sleep -Milliseconds 1000

$proc = Get-Process -Name $AppName -ErrorAction SilentlyContinue
if ($proc) {
    Write-Ok "Running (PID $($proc.Id))."
} else {
    Write-Host "  WARN  Process did not appear within 1s. Check that WebView2 runtime is installed." -ForegroundColor Yellow
    Write-Host "        Download from: https://developer.microsoft.com/en-us/microsoft-edge/webview2/" -ForegroundColor Gray
}

# 9. Claude CLI integration
if (-not $SkipClaudeHooks) {
    Write-Step "Configuring Claude Code integration..."
    Install-ClaudeIntegration -HooksInstallDir $HooksInstallDir
} else {
    Write-Ok "Claude Code integration skipped (-SkipClaudeHooks)."
}

Write-Host "`n$AppName is installed. It will start automatically at every login.`n" -ForegroundColor Green
Write-Host "  To uninstall:  .\uninstall.ps1`n" -ForegroundColor Gray
