param(
    [ValidateSet('all', 'gui', 'cli')]
    [string]$Target = 'all'
)

# Builds antigravity-i18n with the version taken from .\VERSION.
#
# VERSION is the single source of truth. This script stamps it into the Go
# binary with -ldflags, so the CLI, the window title bar and the
# "toolVersion" recorded in antigravity-i18n-state.json all report the same
# number. It does not invent a version: without a VERSION file the build fails
# rather than shipping a binary that misreports itself.

$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot

$versionFile = Join-Path $PSScriptRoot 'VERSION'
if (-not (Test-Path -LiteralPath $versionFile)) {
    throw "VERSION file not found at $versionFile. It is the single source of truth for the release number."
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+([-+][0-9A-Za-z.\-]+)?$') {
    throw "VERSION must be a semantic version (for example 1.0.0), got: '$version'"
}

# Wails reads the PE resource fields from wails.json, and its templating cannot
# see -ldflags, so the same number is written here to keep the compiled binary's
# file version equal to the version the app reports at runtime.
$wailsJson = Join-Path $PSScriptRoot 'wails.json'
$config = Get-Content -LiteralPath $wailsJson -Raw | ConvertFrom-Json
if ($config.info.productVersion -ne $version) {
    $config.info.productVersion = $version
    $json = $config | ConvertTo-Json -Depth 10
    [System.IO.File]::WriteAllText($wailsJson, $json + "`n", (New-Object System.Text.UTF8Encoding($false)))
    Write-Host "wails.json productVersion -> $version"
}

$ldflags = "-X antigravity-i18n/internal/core.Version=$version"

# wails and go are not always on PATH, so resolve them here.
$goBin = 'D:\programs\go\bin'
if (Test-Path -LiteralPath $goBin) { $env:PATH = "$goBin;$env:PATH" }
$env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"

Write-Host "Building antigravity-i18n $version ($Target)"

if ($Target -in @('all', 'gui')) {
    Write-Host '  -> GUI (wails build)'
    & wails build -clean -ldflags $ldflags
    if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }
    $gui = Join-Path $PSScriptRoot 'build\bin\antigravity-i18n.exe'
    $resourceVersion = (Get-Item -LiteralPath $gui).VersionInfo
    if ($resourceVersion.FileVersion -ne $version -or $resourceVersion.ProductVersion -ne $version) {
        throw "Windows file/product version does not match VERSION ($version). Check build/windows/info.json."
    }
    Write-Host ("     {0}  {1:N2} MB" -f $gui, ((Get-Item -LiteralPath $gui).Length / 1MB))
}

if ($Target -in @('all', 'cli')) {
    Write-Host '  -> CLI (go build)'
    & go build -ldflags $ldflags -o 'bin\antigravity-i18n.exe' .
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
    $cli = Join-Path $PSScriptRoot 'bin\antigravity-i18n.exe'
    Write-Host ("     {0}  {1:N2} MB" -f $cli, ((Get-Item -LiteralPath $cli).Length / 1MB))
}

Write-Host "Done: $version"
