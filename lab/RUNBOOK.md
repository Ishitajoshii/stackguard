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

Live evidence still required: the full Codex-to-cloud native capture, hand-inspected explicit mappings, three-credential lineage and final cost reconciliation. The public cloud handoff and [two controlled inventory findings](CONTROLLED_RISKS.md) are complete. Test fixtures are not substitutes for the remaining observations.

The proposed native tracing setup creates a lab KMS key, encrypts and sets seven-day retention on the runtime log group plus `aws/spans` and `/aws/application-signals/data`, and switches the regional X-Ray destination to CloudWatch Logs. Automatic approval review required explicit approval because this includes billable and account-level settings. These changes have not run. The proposal and prior settings are preserved privately. A read-only check of the existing runtime group returned no events; this is not successful native-cloud evidence.
