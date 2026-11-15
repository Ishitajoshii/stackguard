# Lab runbook

## Cloud budget

The assignment's total cloud-spend ceiling is approximately **US$5**. Include model inference, AgentCore runtime, CloudWatch logging/querying, storage and any optional compute in that total.

Before the first paid run, select an available model and region, check their current rates, estimate the run, and set explicit invocation, token, iteration and timeout limits. Track cumulative measured or estimated spend with timestamps. A budget alert is not a guaranteed hard cap; stop optional runs before the remaining balance becomes uncertain.

Do not launch EC2 or persistent storage as an implicit fallback. Use the local machine for the initial lab and the user-selected D:\stackguard-data directory for large files. Terminate unneeded sessions/resources after tests and record cleanup.

The Mumbai Harness is READY and a [public inline-tool round trip](CLOUD_HANDOFF_PROOF.md) passes. Application calls use the `stackguard` assumed caller profile, sourced from the `ishmish` login; the default provisioning profile is not used by the bridge. The execution role is scoped to the deployed runtime, workload identity, managed memory and APAC Nova Micro profile. The [budget estimate](../results/checks/cloud-budget.json) reserves runtime and observability costs; it is not a bill or spending cap. Failed invocations may also incur charges.

Each completed Harness round records the provider's metadata events in the ignored audit log: input, output and total tokens, optional cache read/write counts, and latency. `usage_available=false` means missing telemetry, not zero cost. Preserve individual metadata events rather than assuming repeated events are additive. Failed streams may have incurred charges without returning usage. These counters are not a bill: apply the selected model/region's checked rates, and separately include runtime, logs and storage. The live cost-accounting gate remains open until that reconciliation is performed.

## Identity setup and cleanup checklist

The code can prepare a plan without credentials. For a live run, supply a regional model ID and an execution-role ARN following the current [Harness execution-role guidance](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-security.html). The caller needs both Harness and corresponding Runtime permissions; provisioning additionally needs its documented control-plane permissions and permission to pass the selected role. Use a separately assumed restricted caller profile. Do not broaden IAM to work around an unexplained failure.

Dedicated GitHub and Notion values belong only in the local tool-server environment. The Notion integration must be read-only and shared with the intended page. Never place either value in a model prompt, a cloud-plan file, a benchmark fixture or a committed Codex configuration. Record only fingerprints, purpose and intended binding in the lab inventory.

All plans carry `Project=StackGuard` and `Purpose=credential-tracing-lab` tags. Keep the created harness ARN and region in the ignored deployment export. After the live exercise, stop local processes with Ctrl-C; revoke the dedicated provider tokens; end or let expire the assumed AWS session; delete the exact recorded harness and any lab-created underlying resources/log retention that are no longer needed; then verify their state and record any remaining billed resources. Do not remove unrelated resources based only on a name match. No cleanup action is claimed complete until it is checked against the actual account.

The [full native workflow](NATIVE_SCOUT_PROOF.md), explicit source mappings, three-credential lineage and [two controlled inventory findings](CONTROLLED_RISKS.md) are verified. The [cost checkpoint](../results/checks/cloud-cost-final-estimate.json) records seven attempted cloud sessions, one pre-invocation authentication failure, reported model usage, runtime assumptions and separate KMS/logging reserves. Reported model charges are estimated at USD 0.001604986; the conservative planning scenario including unknown usage and reserves is USD 3.271149221. These are estimates, not a final invoice. The same-day Cost Explorer response remains estimated and cannot isolate all lab charges from account activity.

The user approved native tracing. [Readback](../results/checks/tracing-setup.json) verifies one enabled KMS key, seven-day retention and encryption on the runtime log group, `aws/spans` and `/aws/application-signals/data`, and an ACTIVE CloudWatch Logs trace destination. The key policy limits the regional Logs service to those exact groups. Original pre-change settings and created IDs are preserved privately. AWS reserves the `aws/spans` prefix: X-Ray created that group, after which its key association and retention were verified.

## After the recording/submission

Local capture services are stopped; the Harness, managed memory, roles and tracing resources remain available for the recording. No further paid test is scheduled. KMS storage costs USD 1/month prorated hourly while the key remains billable. Seven-day log retention does not delete the key or restore regional X-Ray routing.

Use the private deployment and `tracing-setup/before.json` receipts to clean up the exact resources. Export/review required evidence first, restore the prior X-Ray destination, and remove only the lab ingestion policy. Review the two shared-named log groups for unrelated spans before deleting or changing them; do not delete unrelated account data. Remove the lab Harness and managed memory when no longer needed. Retain the encryption key until required encrypted logs have been exported or expired, then schedule its deletion using AWS's waiting period and verify the scheduled date. Record remaining billable resources and revoke dedicated tokens only after the recording no longer needs them. Cleanup and final delayed billing are not claimed complete.
