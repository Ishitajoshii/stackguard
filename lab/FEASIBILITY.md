# Feasibility checkpoint

Updated 15 November 2026. Local core, native Codex, authenticated providers and a real cloud inline-tool continuation are demonstrated. The complete native Codex-to-cloud credential-bearing workflow remains open.

| Area | Demonstrated | Remaining |
|---|---|---|
| Build and local tooling | A [fresh tracked-source snapshot](../results/checks/clean-source-local/summary.json) passes builds, the full race suite, vet, four PDF tests, two benchmark-budget tests and original native-capture trace without private inputs. Version inventory: [TOOLCHAIN.md](TOOLCHAIN.md). | Reproduce the real cloud lab from a clean checkout after access is configured. |
| MCP transports | Real repository-file workload: 2,000 calls, 1,000 simultaneous server handlers, zero failures. Original Codex calls pass through both transports with verified file hashes. [Native proof](NATIVE_LOCAL_PROOF.md). | Cloud-selected downstream calls and three owned lab credentials in the real scout workflow. |
| Scale measurements | All 1/4/8/16-worker rows passed: 5,835,736,856-byte scans, 10M-event traces and real MCP calls with 1,000 simultaneous handlers. Canonical hashes agree across workers and both probe memory peaks are below 512,000,000 bytes. [Cross-row verification](../results/checks/full-scale-matrices.json); profiles and extreme session shapes in [SCALING.md](../SCALING.md). | Credential-bearing native graph correctness remains separate. Paused/overlapping times are not uninterrupted speed comparisons; older scan bundles have limited executable provenance. |
| Cancellation | [Large corrected-build proof](../results/bench/late-bounded-interruption/verification.json): exit 17 in 0.153 s, 9,937,372 independently validated partial records and 29 parsed CPU segments. The console-control helper also passes fresh scan, trace, nested-child and real concurrent MCP checks. | Results apply to the recorded workloads; arbitrary OS-blocked filesystem latency is not guaranteed. The original wrapper failure is retained separately from the independently verified probe result. |
| Machine scope | An explicit Codex configuration scan completed with zero errors; private binding details remain local. [Six owned-provider checks](../results/checks/provider-access-retry.json) pass, including the dedicated Notion page. | Provider read success does not establish that token permissions exclude all writes. |
| AWS API shape and identity | Both roles exist, exact resource scoping is verified, and [21 IAM cases pass](aws/README.md). The READY Mumbai Harness uses APAC Nova Micro; [actual streamed continuation](CLOUD_HANDOFF_PROOF.md) passes with a public GitHub read. | Native cloud export and the complete credential-bearing delegation. IAM simulation alone is not live service execution. |
| Native evidence | Installed Codex 0.166.0 emitted original tool-result telemetry for both transports; the adapter passes a fixture cut from that capture. | Actual CloudWatch export, explicit native/proxy/cloud correlation and the credential-bearing expected chain. |
| Scout usefulness | The [authenticated local report](../results/scout/authenticated-local-001/report.md) returns three rechecked issues using Notion preferences and the reviewed skill summary. | Complete Codex-to-cloud scout capture; the local report is not presented as cloud-selected output. |

Reproduce the newest transport observation with:

```powershell
go build -o bin/validated/ ./cmd/...
python scripts/transport-check.py --repo D:/stackguard-data/repos/golang-go --out results/bench/new-transport --workers 4 --bin-dir bin/validated
go test -race ./...
go vet ./...
python -m unittest discover -s lab -p 'test_*.py'
```

Use a fresh output path. The newest original redacted transport records are under [transport-console-helper](../results/bench/transport-console-helper). Earlier unsuccessful attempts and their limits remain documented in SCALING.md; they are not counted as passes.

Live setup commands are in [README](../README.md) and the [runbook](RUNBOOK.md). Five cloud smoke attempts, including failures, are retained in the [cost checkpoint](../results/checks/cloud-cost-checkpoint.json); final billing is not reconciled. The proposed account-level tracing changes await approval and have not run. No secret is required to build, test, render the dossier, inspect the cloud plan, or run local file transport/scale workloads. No raw résumé is sent to the cloud; review the extracted summary first.

Unresolved external decisions: approval of the proposed AWS tracing changes and evaluator acceptance of AgentCore in place of the named cloud option. The user chose to proceed without a precise deadline. The original assignment's allowed-library review is complete: the Go dependencies fit its parser, leak-test and official-cloud-SDK categories; see [DEPENDENCIES.md](DEPENDENCIES.md). Local success does not resolve the outstanding live evidence or cloud-platform acceptance gates.
