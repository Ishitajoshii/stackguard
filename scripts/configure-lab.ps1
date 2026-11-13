param(
 [string]$Profile='profiles/default.yaml',
 [string]$Summary='profiles/summary.json',
 [string]$BinDirectory='bin',
 [ValidatePattern('^[A-Za-z0-9_-]+$')][string]$RunName=('run-'+(Get-Date -Format 'yyyyMMdd-HHmmss')),
 [ValidateRange(1024,65535)][int]$ProxyPort=8765,
 [ValidateRange(1024,65535)][int]$ToolPort=8766,
 [ValidateRange(1024,65535)][int]$CollectorPort=4318
)
$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
function Resolve-LabPath([string]$value) {
 if ([IO.Path]::IsPathRooted($value)) { return [IO.Path]::GetFullPath($value) }
 return [IO.Path]::GetFullPath((Join-Path $root $value))
}
$profilePath=Resolve-LabPath $Profile
$summaryPath=Resolve-LabPath $Summary
if (@($ProxyPort,$ToolPort,$CollectorPort | Select-Object -Unique).Count -ne 3) { throw 'Choose three distinct ports.' }
$binPath=Resolve-LabPath $BinDirectory
$probe=Join-Path $binPath 'nhi-probe.exe'
$scout=Join-Path $binPath 'oss-scout.exe'
if (-not (Test-Path -LiteralPath $probe) -or -not (Test-Path -LiteralPath $scout)) { throw 'Build first: go build -o bin/ ./cmd/...' }
& $scout doctor --profile $profilePath | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Create a valid personal profile with oss-scout init first.' }
$homePath=Join-Path $root "private/$RunName/codex-home"
$capture=Join-Path $root "captures/$RunName"
if ((Test-Path -LiteralPath $homePath) -or (Test-Path -LiteralPath $capture)) { throw 'Choose a fresh RunName; existing evidence is never overwritten.' }
New-Item -ItemType Directory -Path $homePath | Out-Null
$endpoint="http://127.0.0.1:$ProxyPort/mcp"
$arguments=@('proxy','--mode','stdio','--agent','codex','--client','codex-cli','--out',(Join-Path $capture 'local'),'--',$scout,'bridge','--profile',$profilePath,'--summary',$summaryPath,'--endpoint',$endpoint,'--out',(Join-Path $capture 'bridge'))
# JSON string/array quoting is also valid TOML here; no shell command is built.
$commandJson=ConvertTo-Json -InputObject $probe -Compress
$argsJson=ConvertTo-Json -InputObject $arguments -Compress
$config=@"
[mcp_servers.stackguard]
command = $commandJson
args = $argsJson
env_vars = ["AWS_PROFILE", "AWS_REGION"]

[otel]
log_user_prompt = false

[otel.exporter."otlp-http"]
endpoint = "http://127.0.0.1:$CollectorPort/v1/logs"
protocol = "json"
"@
[IO.File]::WriteAllText((Join-Path $homePath 'config.toml'),$config)
[ordered]@{codex_home=$homePath;capture_directory=$capture;profile=$profilePath;summary=$summaryPath;tool_listen="127.0.0.1:$ToolPort";proxy_listen="127.0.0.1:$ProxyPort";proxy_upstream="http://127.0.0.1:$ToolPort/mcp";collector_listen="127.0.0.1:$CollectorPort";note='Configuration only. Start the three services using README, set CODEX_HOME to codex_home, and sign in to Codex there. No credentials were copied and no cloud resource was created.'} | ConvertTo-Json
