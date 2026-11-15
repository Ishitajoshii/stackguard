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
aws configure set role_arn arn:aws:iam::ACCOUNT_ID:role/StackGuardCaller --profile stackguard
aws configure set source_profile stackguard-login --profile stackguard
aws configure set role_session_name stackguard-local --profile stackguard
aws configure set region REGION --profile stackguard
aws sts get-caller-identity --profile stackguard
```

Keep the source profile's `login_session` setting from `aws login`. When its
session expires, repeat `aws login --profile stackguard-login --remote`. The code
belongs in the interactive terminal, never in a tracked file or chat.

The source user needs
[`SignInLocalDevelopmentAccess`](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/SignInLocalDevelopmentAccess.html)
and permission to assume only the caller role. The assumed role does not inherit
the source user's unrelated permissions. No long-lived access keys are needed.
