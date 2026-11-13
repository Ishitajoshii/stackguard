# Scale measurements

Measured on 13 November 2026 on native Windows 11, Intel Core Ultra 5 125H (14 cores / 18 logical processors), Go 1.26.5, with repository/cache/replay data on D:. Hardware and toolchain snapshots are in [lab/ENVIRONMENT.md](lab/ENVIRONMENT.md) and [lab/TOOLCHAIN.md](lab/TOOLCHAIN.md). These are local measurements, not estimates of cloud behavior.

## Real MCP transport

Each row uses 2,000 real calls per mode reading the fetched Go repository README. A server-side barrier proves 1,000 simultaneous handler calls before release; client concurrency alone is not the proof. The driver staggers initial connections by 1 ms to avoid a Windows connection burst, then all 1,000 calls meet the barrier. Separate baseline and proxied server processes use the same workload. Every passing row has zero failures, 2,000 call events and 2,000 separate completion events.

| Workers | Baseline p50 / p99 ms | Proxy p50 / p99 ms | Difference p50 / p99 ms | Probe peak MiB | Ctrl-C seconds |
|---|---:|---:|---:|---:|---:|
| [1](results/bench/transport-w1-final/summary.json) | 261.66 / 1476.56 | 194.05 / 1435.00 | -67.61 / -41.56 | 119.32 | 0.116 |
| [4](results/bench/transport-w4-r5/summary.json) | 425.00 / 1675.97 | 407.02 / 1867.13 | -17.97 / 191.16 | 124.89 | 0.101 |
| [8](results/bench/transport-w8-final/summary.json) | 198.01 / 1341.80 | 449.76 / 1699.58 | 251.75 / 357.79 | 122.86 | 0.147 |
| [16](results/bench/transport-w16-final/summary.json) | 598.35 / 1833.01 | 246.58 / 1329.47 | -351.77 / -503.54 | 125.11 | 0.108 |

Quantiles use the nearest lower order statistic. Differences subtract separate-run quantiles, not paired request overhead. Barrier wait and startup are included: the noisy/negative differences are **not a demonstrated speedup**. Isolated steady-state overhead needs a longer repeated benchmark. Raw baseline/proxy latencies and original proxy records are alongside each linked summary. Original proxy manifests deliberately show incomplete on Ctrl-C, although the driver verifies all requested responses arrived.

Memory uses Windows GetProcessMemoryInfo peak working set (MiB = 1,048,576 bytes), sampled every 20 ms; the API returns an OS-maintained peak. Peak commit is also recorded. These are probe-process figures, excluding the server, load driver and collector. They are not a total-system memory claim.

Earlier connection bursts produced baseline OS error 10061. Staggered connection initiation fixed those runs while preserving the server barrier. A later proxied attempt had 23 HTTP 502s; increasing reusable upstream idle connections from the default small pool to 2,048 removed that failure in subsequent complete runs. Earlier failed attempts are retained locally under ignored .work/bench-attempts. They are not passing measurements.

## Scan and trace worker matrices

Scan corpus: https://github.com/golang/go at commit `224934336832d84eedcf168ae633e376a5decb22`; 16,255 working-tree files, **156,228,793 bytes actually scanned** (about 149 MiB). `.git` is excluded. This is below the 5 GB requirement and does not include the user's machine credential scope. Three candidate fingerprints came from repository content; no provider verification was attempted. Candidate strings in public code may be examples, not valid secrets.

| Workers | Warm scan seconds | Scan peak MiB | 100k trace seconds | Trace peak MiB |
|---|---:|---:|---:|---:|
| 1 | 16.360 | 17.61 | 12.391 | 34.09 |
| 4 | 4.655 | 26.39 | 16.762 | 37.17 |
| 8 | 4.066 | 36.88 | 23.727 | 48.97 |
| 16 | 4.344 | 37.50 | 19.611 | 35.18 |

Raw matrix summaries and CPU/heap profiles: [scan](results/bench/scan-go-warm-matrix), [trace](results/bench/trace-file-100k). The first four-worker scan took 266.526 seconds; its profile was dominated by Windows open/syscall samples. Warm repeats are reported separately and must not be described as a code optimization over that cold run. Windows profiling samples attributed to blocked syscalls are not equivalent to CPU utilization measured by the OS.

The 100,000-event replay is 59,840,000 bytes, derived from the original 4,000 real local-file proxy records. Its provenance manifest hashes both the original JSONL and capture manifest. IDs and parent mappings are rewritten consistently for each copy. **Throughput only:** there are no credential observations in this file workload, so edges/findings/lineage are empty. It exercises normalization, spill sorting, node indexing and event processing, not a large credential graph. Large replay/output files remain on D:, while summary, manifests and profiles are tracked.

Both inventory output and normalized trace/edge/finding output hashes are identical across 1/4/8/16 workers for their respective inputs. [Original 4,000-event trace results](results/bench/trace-original-files/summary.json) are also preserved. This does not replace a hand-checked native Codex/cloud correctness oracle.

## Cancellation and test scope

[Process-level results](results/bench/cancellation-r2/summary.json): scan Ctrl-C 0.072 s, trace 0.051 s, both exit 17 and valid incomplete manifests. These were early interruptions with zero published data records; successful HTTP interruption additionally validates 4,000 already-persisted records. The first cancellation harness attempted to attach before Windows had initialized the console; the runner now waits for the probe's output directory before signalling.

The full Go race suite, vet, four PDF error tests and bounded secret-redaction fuzzing passed locally. Goleak covers proxy and pipeline success/error/cancellation paths. A five-second fuzz command executed 62,304 inputs before its deadline/worker cleanup completed. Algorithm fixtures are labelled as such and are not agent captures.

## Reproduce

```powershell
go build -o bin/ ./cmd/...
python scripts/transport-check.py --repo D:/stackguard-data/repos/golang-go --out results/bench/new-transport --workers 4
python scripts/pipeline-check.py scan --input D:/stackguard-data/repos/golang-go --out results/bench/new-scan --temp D:/stackguard-data/temp
.\bin\sg-bench.exe --replay results/bench/transport-w4-r5/capture/proxy.jsonl --source-manifest results/bench/transport-w4-r5/capture/manifest.json --events 100000 --out D:/stackguard-data/replays/new-replay
python scripts/pipeline-check.py trace --input normalized=D:/stackguard-data/replays/new-replay/replay.jsonl --out D:/stackguard-data/replays/new-trace --temp D:/stackguard-data/temp
python scripts/cancel-check.py --repo D:/stackguard-data/repos/golang-go --events D:/stackguard-data/replays/new-replay/replay.jsonl --temp D:/stackguard-data/temp --out results/bench/new-cancel
go tool pprof -top results/bench/scan-go-warm-matrix/cpu-w4.pprof
```

All destinations must be new. Keep large replays outside the repository. The pipeline runner records memory and hashes, rejects incomplete runs or a peak >=512 MiB, and compares all worker outputs. Queues are bounded to small multiples of workers; external sort chunks are 8 MiB with eight-way merges. More workers need not improve throughput when sorting/disk access is serial or shared.

## Unmet scale gates

- At least 5 GB of public repositories plus selected machine roots. Attempts to shallow-clone Linux and LLVM failed with connection reset/early EOF; only the verified Go corpus is counted.
- Ten million events, including evidence representative of a credential-bearing delegated graph. The replay command supports the count, but 100k is the completed measured run.
- Peak-memory proof for every required workload size, including one extremely large session and many small sessions. Current measurements prove only the stated inputs.
- Native agent correctness and a recorded live demo. Local-file workload evidence cannot establish either.

## Profile-guided optimization

The warm four-worker baseline attributed 71.7% of samples to regexp matching. The focused change skips a provider regex if its mandatory literal prefix is absent and does an allocation-free ASCII precheck for Azure-labelled assignments. Existing detection, chunk-boundary, proxy, trace and race checks passed; an added fuzz property verifies that a matching regex is never rejected by the prefix check.

On the same 156,228,793-byte corpus with four workers, warmed elapsed time decreased from **4.655 s to 1.034 s** (about 77.8% lower in these single runs). Inventory SHA-256 remained `029766ab6f35b2e63d32aec79a7f8a503b225f118addfb140d2c4f8f79cbd316`. Peak working set changed from 26.39 MiB to 38.74 MiB, still below the measured-run budget. These are single-run local comparisons, not a statistical performance guarantee. The first run of the changed binary took 47.357 s and is preserved separately; it is not used as the warm comparison. See [baseline profiles](results/bench/scan-go-warm-matrix) and [after profiles/results](results/bench/scan-go-prefix-warm).

A separate five-second fuzz run over all registered adapters completed 332,397 inputs and verified that successful decodes produce schema-valid events. This validates parser robustness on generated inputs, not native provider schema coverage.
