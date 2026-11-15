# Feasibility checkpoint

Updated 15 November 2026. Local core, native Codex, authenticated providers and the complete native Codex-to-cloud credential-bearing workflow are demonstrated. See the [native proof and disclosed limits](NATIVE_SCOUT_PROOF.md).

| Area | Demonstrated | Remaining |
|---|---|---|
| Build and local tooling | A [fresh tracked-source snapshot](../results/checks/clean-source-local/summary.json) passes builds, the full race suite, vet, four PDF tests, two benchmark-budget tests and original native-capture trace without private inputs. Version inventory: [TOOLCHAIN.md](TOOLCHAIN.md). | Reproduce the real cloud lab from a clean checkout after access is configured. |
| MCP transports | Real file workload: 2,000 calls, 1,000 simultaneous handlers, zero failures. [Native scout](NATIVE_SCOUT_PROOF.md) exercises stdio delegation and cloud-selected downstream HTTP calls with three credential fingerprints. | Human recording/rehearsal. |
| Scale measurements | All 1/4/8/16-worker rows passed: 5,835,736,856-byte scans, 10M-event traces and real MCP calls with 1,000 simultaneous handlers. Canonical hashes agree across workers and both probe memory peaks are below 512,000,000 bytes. [Cross-row verification](../results/checks/full-scale-matrices.json); profiles and extreme session shapes in [SCALING.md](../SCALING.md). | Credential-bearing native graph correctness remains separate. Paused/overlapping times are not uninterrupted speed comparisons; older scan bundles have limited executable provenance. |
| Cancellation | [Large corrected-build proof](../results/bench/late-bounded-interruption/verification.json): exit 17 in 0.153 s, 9,937,372 independently validated partial records and 29 parsed CPU segments. The console-control helper also passes fresh scan, trace, nested-child and real concurrent MCP checks. | Results apply to the recorded workloads; arbitrary OS-blocked filesystem latency is not guaranteed. The original wrapper failure is retained separately from the independently verified probe result. |
| Machine scope | An explicit Codex configuration scan completed with zero errors; private binding details remain local. [Six owned-provider checks](../results/checks/provider-access-retry.json) pass, including the dedicated Notion page. | Provider read success does not establish that token permissions exclude all writes. |
| AWS API shape and identity | Scoped roles, READY Mumbai Harness, APAC Nova Micro, cloud-selected tools and native CloudWatch export all pass live checks. The scoped memory-read correction passes a separate clean native follow-up. | Delayed final billing and post-recording cleanup. |
| Native evidence | Original Codex and 44 CloudWatch spans, explicit result/session/tool-ID links, and a [277-event oracle](../results/checks/native-scout-oracle.json). | Three AWS ingress-parent spans are absent from the export; preserve that limit. |
| Scout usefulness | The [cloud-selected report](../results/scout/native-cloud-20261115/report.md) returns three rechecked issues using the saved profile, Notion preferences and reviewed skills. | Issues can change after the recorded check; recheck before contributing. |

Reproduce the newest transport observation with:

```powershell
go build -o bin/validated/ ./cmd/...
python scripts/transport-check.py --repo D:/stackguard-data/repos/golang-go --out results/bench/new-transport --workers 4 --bin-dir bin/validated
go test -race ./...
go vet ./...
python -m unittest discover -s lab -p 'test_*.py'
```

Use a fresh output path. The newest original redacted transport records are under [transport-console-helper](../results/bench/transport-console-helper). Earlier unsuccessful attempts and their limits remain documented in SCALING.md; they are not counted as passes.

Live setup commands are in [README](../README.md) and the [runbook](RUNBOOK.md). The [latest cost checkpoint](../results/checks/cloud-cost-final-estimate.json) includes failed attempts, the native run and a clean memory follow-up. Tracing is approved and active; final billing and cleanup remain open. No secret is required to build, test, render the dossier, inspect the cloud plan or replay published evidence. No raw résumé is sent to the cloud; review the extracted summary first.

Remaining external steps: evaluator acceptance of AgentCore in place of the named cloud option, the recording/rehearsal and final submission. The user chose to proceed without a precise deadline. The allowed-library review is complete: dependencies fit the parser, leak-test and official-cloud-SDK categories; see [DEPENDENCIES.md](DEPENDENCIES.md).
