#Requires -Version 5.1
<#
.SYNOPSIS
    Installs Locus to run automatically at every Windows login.

.DESCRIPTION
    Builds locus.exe via Wails, copies it to %LOCALAPPDATA%\locus\,
    registers it in HKCU\Software\Microsoft\Windows\CurrentVersion\Run so it
    survives reboots, and starts it immediately.

    Run once. Never think about it again.

    No administrator rights required.

.EXAMPLE
    .\install.ps1
#>

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

function Write-Fail([string]$msg) {
    Write-Host "FAIL  $msg" -ForegroundColor Red
    exit 1
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

# 6. Register in HKCU Run (survives reboots, runs in user session, no admin required)
Write-Step "Registering in Run key..."
Set-ItemProperty -Path $RunKeyPath -Name $AppName -Value "`"$ExeDst`""
$registered = (Get-ItemProperty -Path $RunKeyPath -Name $AppName -ErrorAction SilentlyContinue).$AppName
if (-not $registered) { Write-Fail "Failed to write Run key." }
Write-Ok "Registered: $registered"

# 7. Start immediately
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

Write-Host "`n$AppName is installed. It will start automatically at every login.`n" -ForegroundColor Green
Write-Host "  To uninstall:  .\uninstall.ps1`n" -ForegroundColor Gray
