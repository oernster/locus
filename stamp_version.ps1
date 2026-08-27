# Stamps the version from VERSION into every static file that cannot read it at
# render time.
#
#   ./stamp_version.ps1
#
# VERSION at the repository root is the single place a real version string is
# written down. The Go binary receives it at build time through -ldflags, so it
# needs no stamping. Two kinds of file cannot be given it that way:
#
#   build/windows/info.json   the PE metadata Wails compiles into the executable
#   docs/**/*.html            the GitHub Pages site, served as static files
#
# Each carries a delimited token or a known key; this script overwrites
# whatever sits inside it. Running it against an already-current tree changes
# nothing and prints nothing, so it is safe to run on every build.

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') {
    throw "VERSION does not hold a three-part version string: '$version'"
}

$touched = @()

function Set-Stamped {
    param([string]$Path, [string]$Original, [string]$Updated)
    if ($Updated -ne $Original) {
        [System.IO.File]::WriteAllText($Path, $Updated)
        $script:touched += (Resolve-Path -Relative $Path)
    }
}

# The PE metadata. file_version is a four-part field, so the release version
# takes a trailing zero; ProductVersion is the three-part string as written.
$infoPath = Join-Path $root 'build/windows/info.json'
$info = [System.IO.File]::ReadAllText($infoPath)
$stamped = $info -replace '("file_version"\s*:\s*")[^"]*(")', "`${1}$version.0`${2}"
$stamped = $stamped -replace '("ProductVersion"\s*:\s*")[^"]*(")', "`${1}$version`${2}"
Set-Stamped -Path $infoPath -Original $info -Updated $stamped

# The site. Every page carries the version between the delimiters, so the footer
# states what was actually released rather than what someone last remembered.
Get-ChildItem -Path (Join-Path $root 'docs') -Recurse -Include *.html | ForEach-Object {
    $page = [System.IO.File]::ReadAllText($_.FullName)
    $stampedPage = $page -replace '(<!--VERSION-->)[^<]*(<!--/VERSION-->)', "`${1}$version`${2}"
    Set-Stamped -Path $_.FullName -Original $page -Updated $stampedPage
}

if ($touched.Count -eq 0) {
    Write-Host "Version $version already stamped everywhere."
} else {
    Write-Host "Stamped version $version into:"
    $touched | ForEach-Object { Write-Host "  $_" }
}
