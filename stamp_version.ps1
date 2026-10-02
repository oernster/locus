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
#
# The site's local stylesheet and script links also carry their file's content
# hash, as styles.css?v=<hash>. GitHub Pages lets a browser keep a stylesheet
# for ten minutes, so a fresh page could otherwise be drawn with the old one;
# a changed file is a new address instead.

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

# The asset links. A link names a local file by a relative path; any query it
# already carries is replaced and a fragment is kept. The hash reads CRLF as
# LF, so a Windows checkout and the LF blob GitHub serves agree.
$assetHashLength = 10
$assetLink = '(?<attr>\b(?:href|src)=)(?<quote>["''])(?<path>[^"''?#]+\.(?:css|js))(?:\?[^"''#]*)?(?<fragment>#[^"'']*)?\k<quote>'
$linked = @()

function Get-AssetHash {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "A site page links $Path, which does not exist; nothing hashed."
    }
    # Latin-1 maps every byte to one character, so the CRLF swap is exact.
    $latin1 = [System.Text.Encoding]::GetEncoding('iso-8859-1')
    $bytes = $latin1.GetBytes($latin1.GetString([System.IO.File]::ReadAllBytes($Path)).Replace("`r`n", "`n"))
    $digest = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
    (-join ($digest | ForEach-Object { $_.ToString('x2') })).Substring(0, $assetHashLength)
}

function Add-AssetHashes {
    param([string]$Page, [string]$Directory)
    [regex]::Replace($Page, $assetLink, {
        param($link)
        $path = $link.Groups['path'].Value
        # Remote, protocol-relative and root-absolute links are not this site's files.
        if ($path.StartsWith('/') -or $path.Contains(':')) {
            return $link.Value
        }
        $quote = $link.Groups['quote'].Value
        '{0}{1}{2}?v={3}{4}{1}' -f $link.Groups['attr'].Value, $quote, $path,
            (Get-AssetHash (Join-Path $Directory $path)), $link.Groups['fragment'].Value
    })
}

Get-ChildItem -Path (Join-Path $root 'docs') -Recurse -Include *.html | ForEach-Object {
    $page = [System.IO.File]::ReadAllText($_.FullName)
    $linkedPage = Add-AssetHashes -Page $page -Directory $_.DirectoryName
    if ($linkedPage -ne $page) {
        [System.IO.File]::WriteAllText($_.FullName, $linkedPage)
        $linked += (Resolve-Path -Relative $_.FullName)
    }
}

if ($touched.Count -eq 0 -and $linked.Count -eq 0) {
    Write-Host "Version $version and asset hashes already stamped everywhere."
}
if ($touched.Count -gt 0) {
    Write-Host "Stamped version $version into:"
    $touched | ForEach-Object { Write-Host "  $_" }
}
if ($linked.Count -gt 0) {
    Write-Host "Stamped asset hashes into:"
    $linked | ForEach-Object { Write-Host "  $_" }
}
