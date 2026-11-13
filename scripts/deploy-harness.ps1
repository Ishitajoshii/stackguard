param(
 [Parameter(Mandatory=$true)][string]$InputFile,
 [Parameter(Mandatory=$true)][string]$Region,
 [string]$Profile='default',
 [Parameter(Mandatory=$true)][string]$OutputFile
)
$ErrorActionPreference='Stop'
if (Test-Path -LiteralPath $OutputFile) { throw 'Choose a new private output file.' }
$plan = Get-Content -LiteralPath $InputFile -Raw | ConvertFrom-Json
if (-not $plan.model -or -not $plan.executionRoleArn -or $plan.maxTokens -gt 2048 -or $plan.maxIterations -gt 6 -or $plan.allowedTools -contains '*') { throw 'Review explicit model, role, allowlist and bounded limits first.' }
# No credentials or provider content are printed. This is an explicit deployment
# command; it is not called by setup, doctor, tests or profile preparation.
$createdText = & aws bedrock-agentcore-control create-harness --profile $Profile --region $Region --cli-input-json ('file://' + (Resolve-Path -LiteralPath $InputFile).Path) --output json 2>$null
if ($LASTEXITCODE -ne 0) { throw 'CreateHarness failed; check role/model/region permissions in AWS.' }
$created = $createdText | ConvertFrom-Json
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
