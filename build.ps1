# Builds Locus from VERSION.
#
#   ./build.ps1        stamp the version, then build build/bin/locus.exe
#
# The version reaches the binary two ways and both start at VERSION: the static
# files are stamped by stamp_version.ps1; the Go string is set here with
# -ldflags. Nothing else may write a version down.
#
# Note that -X only reaches a var. Against a const it silently does nothing and
# the binary ships the sentinel with no error to read, which is why
# internal/version.Version is declared as a var and must stay one.

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
Write-Host "Building Locus $version"

# Calling a .ps1 does not set $LASTEXITCODE. It stays at whatever the last
# native command left; in a fresh session it is empty. Both scripts run under
# $ErrorActionPreference = 'Stop', so a failure in the stamp throws and stops
# this one; there is nothing to test afterwards.
& (Join-Path $root 'stamp_version.ps1')

Push-Location $root
try {
    & wails build -ldflags "-X github.com/oernster/locus/internal/version.Version=$version"
    if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host "Built build/bin/locus.exe at version $version"
