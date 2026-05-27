#Requires -Version 5.1
<#
.SYNOPSIS
    Removes Locus from Windows startup and optionally purges all data.

.DESCRIPTION
    Stops any running Locus process, removes the HKCU Run key that causes
    it to launch at login, and deletes the install directory
    (%LOCALAPPDATA%\locus\).

    By default the task database (%APPDATA%\locus\locus.db) is kept so your
    board, sessions, outcomes, and snapshots survive a reinstall.
    Pass -PurgeData to delete it too.

    No administrator rights required.

.PARAMETER PurgeData
    Also removes %APPDATA%\locus\ including the task database and all
    snapshots. All board history will be permanently lost.

.EXAMPLE
    .\uninstall.ps1
    .\uninstall.ps1 -PurgeData
#>
param(
    [switch]$PurgeData
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

# 4. Optionally purge data
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
