# Scale proof

Status: no scale benchmark has run. Initial CLI race tests are a toolchain preflight, not scale evidence.

## Required workloads

| Stage | Target | Status |
|---|---|---|
| scan | At least 5 GB public repositories plus machine scope | Pending |
| proxy | At least 1,000 simultaneous real tools/call requests | Pending |
| trace | Replay original captures up to 10 million events | Pending; original captures required first |
| worker matrix | 1, 4, 8 and 16 workers | Pending |
| memory | Peak below 512 MB for every required run | Pending |
| optimization | Largest-run CPU/heap profiles and measured before/after fix | Pending |

Large data directories are reserved under D:\stackguard-data. Record input manifests, actual concurrency, errors, throughput, memory measurement boundaries and commands for every run. Preserve raw latency evidence for direct/proxy p50/p99 comparisons.

Separate original agent captures from replay data. Measure correctness on original captures, and report probe memory separately from the driver, servers and collectors. Never fill this document with estimated results presented as measurements.
