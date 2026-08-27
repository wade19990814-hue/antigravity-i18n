param(
    [switch]$CheckOnly
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$PackageName = 'antigravity-i18n'
$Version = '2.0.0'
$Tag = 'v2.0.0'
$Repository = 'wade19990814-hue/antigravity-i18n'
$OldPackage = 'antigravity-zh'
$OldPackageNotice = 'Package renamed to antigravity-i18n. Use: npx antigravity-i18n apply --locale <code>'

$RepoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Set-Location -LiteralPath $RepoRoot

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)]
        [string]$File,
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Arguments
    )

    & $File @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$File failed with exit code $LASTEXITCODE."
    }
}

function Test-NpmVersionPublished {
    $result = & npm view "$PackageName@$Version" version --json 2>$null
    return $LASTEXITCODE -eq 0 -and ($result -join '').Trim('"', [char]13, [char]10) -eq $Version
}

Write-Host "==> Preflight for $PackageName@$Version"

foreach ($command in 'git', 'node', 'npm', 'gh') {
    if (-not (Get-Command $command -ErrorAction SilentlyContinue)) {
        throw "Required command is unavailable: $command"
    }
}

$package = Get-Content -LiteralPath (Join-Path $RepoRoot 'package.json') -Raw -Encoding UTF8 | ConvertFrom-Json
if ($package.name -ne $PackageName -or $package.version -ne $Version) {
    throw "package.json must contain $PackageName@$Version."
}

$branch = (& git branch --show-current).Trim()
if ($LASTEXITCODE -ne 0 -or $branch -ne 'main') {
    throw "Release must run from main; current branch is '$branch'."
}

$dirty = (& git status --porcelain) -join ''
if ($LASTEXITCODE -ne 0 -or $dirty) {
    throw 'Working tree is not clean.'
}

Invoke-Checked git fetch origin --prune --tags

$parity = ((& git rev-list --left-right --count HEAD...origin/main) -join ' ') -split '\s+'
if ($LASTEXITCODE -ne 0 -or $parity.Count -lt 2 -or $parity[0] -ne '0' -or $parity[1] -ne '0') {
    throw "main and origin/main are not synchronized: $($parity -join ' ')."
}

$npmUser = (& npm whoami 2>$null).Trim()
if ($LASTEXITCODE -ne 0 -or -not $npmUser) {
    throw 'npm is not logged in. Run: npm login --auth-type=web'
}
Write-Host "    npm user: $npmUser"

Invoke-Checked gh auth status
Invoke-Checked npm test
Invoke-Checked npm audit --audit-level=high

if ($CheckOnly) {
    Write-Host '==> Check-only mode passed. No package, tag, or release was changed.'
    exit 0
}

if (Test-NpmVersionPublished) {
    Write-Host "==> npm already contains $PackageName@$Version; skipping publish."
} else {
    Write-Host "==> Publishing $PackageName@$Version to npm"
    Invoke-Checked npm publish --access public
}

$published = $false
for ($attempt = 1; $attempt -le 6; $attempt += 1) {
    if (Test-NpmVersionPublished) {
        $published = $true
        break
    }
    Start-Sleep -Seconds 5
}
if (-not $published) {
    throw "npm registry did not confirm $PackageName@$Version."
}
Write-Host "    npm verified: https://www.npmjs.com/package/$PackageName/v/$Version"

Write-Host "==> Deprecating the old package name"
Invoke-Checked npm deprecate "$OldPackage@*" $OldPackageNotice

$head = (& git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) {
    throw 'Could not resolve HEAD.'
}

$localTagCommit = (& git rev-list -n 1 $Tag 2>$null)
if ($LASTEXITCODE -eq 0 -and $localTagCommit) {
    if ($localTagCommit.Trim() -ne $head) {
        throw "$Tag already points to a different commit."
    }
    Write-Host "==> Local tag $Tag already points to HEAD."
} else {
    Invoke-Checked git tag -a $Tag -m "$PackageName $Version"
}

$remoteTag = (& git ls-remote --tags origin "refs/tags/$Tag") -join ''
if ($LASTEXITCODE -ne 0) {
    throw 'Could not inspect remote tags.'
}
if ($remoteTag) {
    Write-Host "==> Remote tag $Tag already exists."
} else {
    Invoke-Checked git push origin $Tag
}

$releaseNotes = @'
## Highlights

- Eight bundled language packs: Simplified Chinese, Japanese, Korean, Spanish, German, French, Brazilian Portuguese, and Russian.
- Generic locale CLI with direct language switching and official English restoration.
- Byte-exact restore, verified clean backups, atomic archive replacement, and archive identity checks.
- CLDR plural selection, Russian one/few/many forms, and right-to-left layout support.
- Complete localized README set and validated source-equal cognate handling.
- Verified against an Antigravity 2.10.0 archive sandbox.

## Install

    npx antigravity-i18n apply --locale zh-CN
    npx antigravity-i18n apply --locale ja
    npx antigravity-i18n apply --locale ko
    npx antigravity-i18n apply --locale es
    npx antigravity-i18n apply --locale de
    npx antigravity-i18n apply --locale fr
    npx antigravity-i18n apply --locale pt-BR
    npx antigravity-i18n apply --locale ru

Restore the official English build:

    npx antigravity-i18n restore

## Migration

The npm package and CLI were renamed from antigravity-zh to antigravity-i18n. Existing zh and en shorthand commands remain supported.
'@

$existingRelease = & gh release view $Tag --repo $Repository --json tagName 2>$null
if ($LASTEXITCODE -eq 0 -and $existingRelease) {
    Write-Host "==> GitHub Release $Tag already exists."
} else {
    Invoke-Checked gh release create $Tag --repo $Repository --title $Tag --notes $releaseNotes --verify-tag
}

Invoke-Checked gh repo edit $Repository --homepage "https://www.npmjs.com/package/$PackageName"

Write-Host '==> Final verification'
Invoke-Checked npm view "$PackageName@$Version" version
Invoke-Checked npm view $OldPackage deprecated
Invoke-Checked gh release view $Tag --repo $Repository --json name,tagName,publishedAt,url
Invoke-Checked gh repo view $Repository --json description,homepageUrl,latestRelease,url

$finalDirty = (& git status --porcelain) -join ''
if ($LASTEXITCODE -ne 0 -or $finalDirty) {
    throw 'Release finished, but the working tree is no longer clean.'
}

Write-Host ''
Write-Host "Released $PackageName@$Version successfully."
Write-Host "npm:    https://www.npmjs.com/package/$PackageName"
Write-Host "GitHub: https://github.com/$Repository/releases/tag/$Tag"

