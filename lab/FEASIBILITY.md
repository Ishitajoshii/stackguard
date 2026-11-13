# Feasibility checkpoint

Updated 13 November 2026. Local core and protocol feasibility is demonstrated; cloud/native-agent feasibility remains open. Credentials are intentionally deferred by the user, not replaced by test credentials.

| Area | Demonstrated | Remaining |
|---|---|---|
| Build and local tooling | Native Windows builds; Go race suite, vet, leak checks and four local PDF tests pass. Version inventory: [TOOLCHAIN.md](TOOLCHAIN.md). | Reproduce the real cloud lab from a clean checkout after access is configured. |
| MCP transports | Real repository-file workload: 2,000 calls, 1,000 simultaneous server handlers, zero failures, matching call/completion counts. Stdio child-process tests cover control/data/error/cancellation frames. | Original Codex call through stdio and cloud-selected downstream calls through HTTP. |
| Scale input | A complete sixteen-worker scan covered 5,835,736,856 repository bytes with zero errors and 104.13 MiB peak working set. A replay contains exactly 10,000,000 records and 5,984,000,000 bytes, with a checked SHA-256. | Remaining full-corpus scan workers are running; the optimized 10M trace is queued. Generation is not proof of trace throughput. |
| Machine scope | An explicit Codex configuration scan completed with zero errors; private binding details remain local. | Owned lab token registration and valid-provider checks after tokens are supplied. |
| AWS API shape | Official Go SDK v1.58.0 supports the required Harness stream/inline continuation contract. [API check](AGENTCORE_API_CHECK.md). | Assumed caller role, execution role, permitted regional model, ready deployment and actual streamed continuation. |
| Native evidence | Adapters/collector compile and pass contract tests. | Capture installed Codex telemetry and an actual CloudWatch export, inspect identifiers and compare the expected chain. |
| Scout usefulness | Anonymous public API smoke calls and bounded selection/reporting logic. | Authenticated useful shortlist with suitable rechecked issues, Notion preferences and a human-reviewed local skill summary. |

Reproduce the newest transport observation with:

```powershell
go build -o bin/validated/ ./cmd/...
python scripts/transport-check.py --repo D:/stackguard-data/repos/golang-go --out results/bench/new-transport --workers 4 --bin-dir bin/validated
go test -race ./...
go vet ./...
python -m unittest discover -s lab -p 'test_*.py'
```

Use a fresh output path. Original redacted transport records are under [transport-cancellation-fixes-r2](../results/bench/transport-cancellation-fixes-r2). The first attempt in this revision exited before producing a proxied load report; it is preserved under ignored .work/bench-attempts and is not a pass. The second attempt is the reported measurement. No root cause is asserted for the first exit.

Live setup commands are in [README](../README.md) and the [runbook](RUNBOOK.md). No cloud deployment or paid invocation has been performed. No secret is required to build, test, render the dossier, inspect the cloud plan, or run local file transport/scale workloads. No raw résumé is sent to the cloud; review the extracted summary first.

Unresolved external decisions: exact submission cut-off and evaluator acceptance of AgentCore in place of the named cloud option. The original assignment's allowed-library review is complete: the Go dependencies fit its parser, leak-test and official-cloud-SDK categories; see [DEPENDENCIES.md](DEPENDENCIES.md). Local success does not resolve the outstanding live evidence or cloud-platform acceptance gates.
