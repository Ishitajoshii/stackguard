# Feasibility checkpoint

Updated 14 November 2026. Local core, native Codex and protocol feasibility are demonstrated; cloud workflow feasibility remains open. Credentials are intentionally deferred by the user, not replaced by test credentials.

| Area | Demonstrated | Remaining |
|---|---|---|
| Build and local tooling | A [fresh tracked-source snapshot](../results/checks/clean-source-local/summary.json) passes builds, the full race suite, vet, four PDF tests, two benchmark-budget tests and original native-capture trace without private inputs. Version inventory: [TOOLCHAIN.md](TOOLCHAIN.md). | Reproduce the real cloud lab from a clean checkout after access is configured. |
| MCP transports | Real repository-file workload: 2,000 calls, 1,000 simultaneous server handlers, zero failures. Original Codex calls pass through both transports with verified file hashes. [Native proof](NATIVE_LOCAL_PROOF.md). | Cloud-selected downstream calls and three owned lab credentials in the real scout workflow. |
| Scale input | Sixteen- and eight-worker scans covered 5,835,736,856 repository bytes with zero errors and identical inventory hashes. Main, one-session and ten-million-session replay inputs have verified counts and SHA-256. | Remaining full-corpus scan workers and the optimized 10M trace matrix/extreme-session runs are in progress. Generation is not proof of trace throughput. |
| Machine scope | An explicit Codex configuration scan completed with zero errors; private binding details remain local. | Owned lab token registration and valid-provider checks after tokens are supplied. |
| AWS API shape | Official Go SDK v1.58.0 supports the required Harness stream/inline continuation contract. [API check](AGENTCORE_API_CHECK.md). | Assumed caller role, execution role, permitted regional model, ready deployment and actual streamed continuation. |
| Native evidence | Installed Codex 0.166.0 emitted original tool-result telemetry for both transports; the adapter passes a fixture cut from that capture. | Actual CloudWatch export, explicit native/proxy/cloud correlation and the credential-bearing expected chain. |
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
