param(
 [Parameter(Mandatory=$true)][string]$InputFile,
 [Parameter(Mandatory=$true)][string]$Region,
 [string]$Profile='default',
 [Parameter(Mandatory=$true)][string]$OutputFile
)
$ErrorActionPreference='Stop'
foreach ($path in @($OutputFile, ($OutputFile+'.created.json'), ($OutputFile+'.error.txt'))) {
 if (Test-Path -LiteralPath $path) { throw 'Choose a new private output file.' }
}
$plan = Get-Content -LiteralPath $InputFile -Raw | ConvertFrom-Json
$lifecycle = $plan.environment.agentCoreRuntimeEnvironment.lifecycleConfiguration
if (-not $plan.model -or -not $plan.executionRoleArn -or $plan.maxTokens -lt 1 -or $plan.maxTokens -gt 2048 -or $plan.maxIterations -lt 1 -or $plan.maxIterations -gt 6 -or $plan.timeoutSeconds -lt 1 -or $plan.timeoutSeconds -gt 60 -or -not $plan.allowedTools -or $plan.allowedTools -contains '*' -or $lifecycle.idleRuntimeSessionTimeout -ne 60 -or $lifecycle.maxLifetime -ne 300) { throw 'Review explicit model, role, allowlist and bounded invocation/session limits first.' }
# No credentials or provider content are printed. This is an explicit deployment
# command; it is not called by setup, doctor, tests or profile preparation.
$createdText = & aws bedrock-agentcore-control create-harness --profile $Profile --region $Region --cli-input-json ('file://' + (Resolve-Path -LiteralPath $InputFile).Path) --output json 2>($OutputFile+'.error.txt')
if ($LASTEXITCODE -ne 0) { throw 'CreateHarness failed; check role/model/region permissions in AWS.' }
$created = $createdText | ConvertFrom-Json
$createdText | Set-Content -LiteralPath ($OutputFile+'.created.json')
$deadline = [DateTime]::UtcNow.AddMinutes(5)
while ([DateTime]::UtcNow -lt $deadline) {
 $stateText = & aws bedrock-agentcore-control get-harness --profile $Profile --region $Region --harness-id $created.harnessId --output json 2>$null
 if ($LASTEXITCODE -ne 0) { throw 'GetHarness failed; the created resource may need cleanup.' }
 $state = $stateText | ConvertFrom-Json
 if ($state.status -eq 'READY') {
  $stateText | Set-Content -LiteralPath $OutputFile
  Write-Output 'Harness READY. Configure its ARN locally, then run the explicit cloud smoke test.'
  exit 0
 }
 if ($state.status -match 'FAILED') { throw 'Harness provisioning failed; inspect the resource in AWS and clean it up.' }
 Start-Sleep -Seconds 5
}
throw 'Harness readiness timeout; check and clean up the tagged resource before retrying.'
