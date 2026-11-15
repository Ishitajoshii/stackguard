# AWS role setup

These JSON files are templates, not deployable account configuration. Replace
`${AccountId}`, `${Region}`, `${HarnessName}` and `${CallerUserArn}` in copies under
ignored `private/`. Never put access keys or login tokens in these files.

The local bridge uses `StackGuardCaller`, assumed from a separate browser-login
profile. Its policy allows invocation of one named harness family, backing runtimes
tagged `Project=StackGuard` in the same account/region, and STS identity inspection.
It grants no IAM administration, direct model calls, S3 access, or runtime command
execution. Once deployed, verify backing-runtime tags and replace the harness and
runtime patterns with their exact ARNs. The caller trust names the specific IAM
user, not the account root.

The caller action set was generated from `lab/cloud/harness.go` with IAM Policy
Autopilot 0.4.0, then its generated resource patterns were narrowed to the named
harness and tagged backing runtimes. AWS documents both invoke actions as required on the harness ARN:
[Harness IAM permissions](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-security.html#harness-iam-permissions).
`GetCallerIdentity` has no resource-level scope.
The IAM simulator evaluates `InvokeAgentRuntime` against runtime resources, so
test that permission separately, including a missing/wrong-tag denial. A simulator
result on a harness ARN alone does not establish the dependent runtime permission.

Use a provisioning identity to create the roles and attach the reviewed policies.
Do not grant provisioning permissions to the application's caller role. Validate
the materialized identity policies with IAM Access Analyzer before attachment,
then simulate allowed and denied resources/actions and verify actual role
assumption with STS. Simulation does not prove live Harness/model execution.

After browser login, configure the application's separate assumed-role profile:

```powershell
# Confirm aws.exe resolves to your trusted AWS CLI installation on PATH.
Get-Command aws.exe -CommandType Application
aws configure set credential_process 'aws.exe configure export-credentials --profile stackguard-login --format process' --profile stackguard-source
aws configure set region REGION --profile stackguard-source
aws configure set role_arn arn:aws:iam::ACCOUNT_ID:role/StackGuardCaller --profile stackguard
aws configure set source_profile stackguard-source --profile stackguard
aws configure set role_session_name stackguard-local --profile stackguard
aws configure set region REGION --profile stackguard
aws sts get-caller-identity --profile stackguard
```

Keep the source profile's `login_session` setting from `aws login`. When its
session expires, repeat `aws login --profile stackguard-login --remote`. The code
belongs in the interactive terminal, never in a tracked file or chat.

The pinned Go config module v1.33.18 recognizes `login_session` but omits it from
its source-profile `hasCredentials` check. Directly chaining the role to the login
profile failed locally even though AWS CLI STS succeeded. The separate
`stackguard-source` profile uses AWS's documented
[credential-process compatibility path](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sign-in.html).
Only the SDK/CLI consumes that process's credential output; do not run the export
command manually or redirect its output into evidence. The same `stackguard`
profile works for the CLI and the Go app without changing pinned dependencies.
The [Go SDK STS check](../../results/checks/aws-roles/sdk-login.json) passed after
this workaround. On Windows, use the trusted `aws.exe` on PATH: the pinned Go
process provider failed to execute a quoted executable path containing spaces.

The source user needs
[`SignInLocalDevelopmentAccess`](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/SignInLocalDevelopmentAccess.html)
and permission to assume only the caller role. The assumed role does not inherit
the source user's unrelated permissions. No long-lived access keys are needed.

## Execution role

`StackGuardHarnessExecution` trusts the AgentCore service only for this account's
harness/runtime resources in the selected region. The bootstrap trust is broader
than a single harness because the service-generated IDs do not yet exist. Narrow
both trust ARN patterns and log resources after deployment. Users cannot assume
this service role directly.

Also replace `${InferenceProfileId}` and `${ModelId}` in `execution-policy.json`.
The model permission requires the selected inference-profile ARN; direct model
calls without that profile are denied. The model's region wildcard supports its
cross-region routing, while the condition confines it to the selected profile.
The initial lab choice is `apac.amazon.nova-micro-v1:0` from Mumbai. This profile
can route within APAC; it is not Mumbai-only inference. Listing the profile does
not prove billing eligibility or successful inference.

The remaining permissions support the public managed container, scoped runtime
logs, regional tracing/metrics, and the named harness workload identity. Token
and telemetry APIs that do not support resource-level permissions use `*`, with
service, namespace or region conditions where applicable. There are no S3,
Secrets Manager, IAM-administration, browser, code-interpreter, or role-chaining
grants. Logging resource-policy administration is intentionally not delegated to
the execution role; any required observability setup belongs to provisioning.

Sources checked 2026-11-15:

- [Harness execution role](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-security.html#harness-execution-role-policy)
- [Runtime confused-deputy protection](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/runtime-security-best-practices.html)
- [Inference-profile conditions](https://docs.aws.amazon.com/bedrock/latest/userguide/inference-profiles-prereq.html)

## Read-only verification

Run using a provisioning profile with IAM simulation permissions. This script
uses the AWS CLI and prints no credential values. Output must be a fresh path.

```powershell
python scripts/verify-aws-roles.py --account ACCOUNT_ID --region REGION --scope all --out private/aws-role-checks.json
```

The [caller checks](../../results/checks/aws-roles/caller.json) verify actual STS
assumption plus nine policy cases. The
[execution checks](../../results/checks/aws-roles/execution.json) cover twelve
allowed/denied permission cases. These checks do not create a harness, invoke a
model, test actual service-role assumption, or establish native cloud evidence.
Keep the runbook's pricing/budget check before the first paid run.

The live lab is now [READY](../../results/checks/harness-ready.json). Its caller,
execution trust, workload identity and log permissions were narrowed to the actual
resource ARNs returned by AWS; these private IDs are not embedded in templates.
All [21 checks passed again](../../results/checks/aws-roles/exact-resources.json).
When verifying these tightened policies, also pass `--harness-arn` and
`--runtime-arn` with the observed deployed ARNs. Without those arguments, the
verifier exercises the earlier bootstrap name/tag scope instead.

The first live invoke revealed two additional service requirements. Execution
trust must include the backing runtime's endpoint/version suffixes: scope
`aws:SourceArn` to the observed runtime ARN plus `*`, retaining the source-account
condition. The bare runtime ARN passed static validation but service assumption
failed. Also attach `memory-policy.json` after substituting the actual managed
memory ARN from `GetHarness`: even the default managed-memory configuration reads
and writes session events. The full native run also exposed the documented
`RetrieveMemoryRecords` requirement; the scoped fix passes a fresh native follow-up
with no error spans. Never grant these actions against all memory resources.
