# Feasibility checkpoint

Updated 14 November 2026. Local core, native Codex and protocol feasibility are demonstrated; cloud workflow feasibility remains open. Credentials are intentionally deferred by the user, not replaced by test credentials.

| Area | Demonstrated | Remaining |
|---|---|---|
| Build and local tooling | A [fresh tracked-source snapshot](../results/checks/clean-source-local/summary.json) passes builds, the full race suite, vet, four PDF tests, two benchmark-budget tests and original native-capture trace without private inputs. Version inventory: [TOOLCHAIN.md](TOOLCHAIN.md). | Reproduce the real cloud lab from a clean checkout after access is configured. |
| MCP transports | Real repository-file workload: 2,000 calls, 1,000 simultaneous server handlers, zero failures. Original Codex calls pass through both transports with verified file hashes. [Native proof](NATIVE_LOCAL_PROOF.md). | Cloud-selected downstream calls and three owned lab credentials in the real scout workflow. |
| Scale measurements | All 1/4/8/16-worker rows passed: 5,835,736,856-byte scans, 10M-event traces and real MCP calls with 1,000 simultaneous handlers. Canonical hashes agree across workers and both probe memory peaks are below 512,000,000 bytes. [Cross-row verification](../results/checks/full-scale-matrices.json); profiles and extreme session shapes in [SCALING.md](../SCALING.md). | Credential-bearing native graph correctness remains separate. Paused/overlapping times are not uninterrupted speed comparisons; older scan bundles have limited executable provenance. |
| Cancellation | [Large corrected-build proof](../results/bench/late-bounded-interruption/verification.json): exit 17 in 0.153 s, 9,937,372 independently validated partial records and 29 parsed CPU segments. The console-control helper also passes fresh scan, trace, nested-child and real concurrent MCP checks. | Results apply to the recorded workloads; arbitrary OS-blocked filesystem latency is not guaranteed. The original wrapper failure is retained separately from the independently verified probe result. |
| Machine scope | An explicit Codex configuration scan completed with zero errors; private binding details remain local. | Owned lab token registration and valid-provider checks after tokens are supplied. |
| AWS API shape and identity | Official Go SDK v1.58.0 supports the required Harness stream/inline continuation contract. [API check](AGENTCORE_API_CHECK.md). Browser CLI login and the assumed caller role are verified; both roles exist and [21 IAM policy cases pass](aws/README.md). The Nova Micro APAC profile is listed from Mumbai. | Live model eligibility/inference, ready deployment, exact service-resource scoping and actual streamed continuation. IAM simulation is not live service execution. |
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

Use a fresh output path. The newest original redacted transport records are under [transport-console-helper](../results/bench/transport-console-helper). Earlier unsuccessful attempts and their limits remain documented in SCALING.md; they are not counted as passes.

Live setup commands are in [README](../README.md) and the [runbook](RUNBOOK.md). No cloud deployment or paid invocation has been performed. No secret is required to build, test, render the dossier, inspect the cloud plan, or run local file transport/scale workloads. No raw résumé is sent to the cloud; review the extracted summary first.

Unresolved external decisions: exact submission cut-off and evaluator acceptance of AgentCore in place of the named cloud option. The original assignment's allowed-library review is complete: the Go dependencies fit its parser, leak-test and official-cloud-SDK categories; see [DEPENDENCIES.md](DEPENDENCIES.md). Local success does not resolve the outstanding live evidence or cloud-platform acceptance gates.
