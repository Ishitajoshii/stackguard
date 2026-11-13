param(
    [Parameter(Mandatory=$true)][string]$Message,
    [Parameter(Mandatory=$true)][string[]]$Paths,
    [string[]]$Packages = @('./...'),
    [string]$Todo = ''
)
$ErrorActionPreference = 'Stop'
$env:GOCACHE = 'D:\stackguard-data\go-cache'
$env:GOMODCACHE = 'D:\stackguard-data\go-mod'
$env:TEMP = 'D:\stackguard-data\temp'
$env:TMP = $env:TEMP
& go test -race @Packages
if ($LASTEXITCODE -ne 0) { throw 'Tests failed; nothing committed.' }
& git diff --check
if ($LASTEXITCODE -ne 0) { throw 'Whitespace check failed; nothing committed.' }
& git add -- @Paths
if ($LASTEXITCODE -ne 0) { throw 'Staging failed.' }
& git commit -m $Message
if ($LASTEXITCODE -ne 0) { throw 'Commit failed.' }
$hash = (& git rev-parse --short HEAD).Trim()
if ($Todo) {
    $todoPath = Join-Path $PSScriptRoot '../TODO.md'
    $content = [IO.File]::ReadAllText($todoPath)
    $old = '- [ ] ' + $Todo
    if (-not $content.Contains($old)) { throw "Commit $hash exists, but TODO entry was not found. Update it manually." }
    $content = $content.Replace($old, ('- [x] ' + $Todo + ' Evidence: ' + $hash + '; targeted race tests passed.'))
    [IO.File]::WriteAllText($todoPath, $content)
}
Write-Output "Completed $hash"
