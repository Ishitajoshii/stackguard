# Local environment evidence

Observed on 13 November 2026. Capacity values are a snapshot, not reserved resources.

| Item | Observation |
|---|---|
| OS | Windows 11 Home Single Language, 10.0.26200 |
| Architecture | amd64 |
| Visible RAM | 16,373,112 KiB (about 15.6 GiB) |
| Available RAM at observation | 1,876,968 KiB (about 1.8 GiB) |
| C: free | 1,052,479,488 bytes (about 1.0 GiB) |
| D: free | 193,120,010,240 bytes (about 179.9 GiB) |
| E: free | 322,326,441,984 bytes (about 300.2 GiB) |

Collection: `Get-CimInstance Win32_OperatingSystem` and `Get-PSDrive -PSProvider FileSystem` in PowerShell. No hardware serial numbers, machine names or account identifiers are needed in the evidence.

C: cannot hold the required 5 GB repository corpus plus replay and profiling files. Select a separate data directory on D: or E:, or free sufficient C: space before those tests. The user-selected D:\stackguard-data now holds public repositories, build caches, temporary files and throughput replays. A complete system scan must explicitly record its selected roots and exclusions.

The available RAM snapshot also reflects other running applications. Measure the probe's peak process memory during each real benchmark; this inventory does not establish the 512 MB acceptance criterion.

## Execution environment decision

Continue on native Windows. The initial Go binary builds, ordinary/race tests run with the installed GCC, and the local Python helper extracts the real PDF successfully. AWS CLI and Codex CLI are available. No demonstrated dependency currently requires WSL.

Use the user-selected `D:\stackguard-data` tree for repositories, replays, profiles, Go build cache and temporary test files. The directories have been created, but no benchmark corpus has been downloaded. Cloud access is still a separate feasibility gate. Revisit WSL only if a concrete integration failure requires it.

Benchmark CPU: Intel Core Ultra 5 125H, 14 cores and 18 logical processors (Win32_Processor observation, 13 November 2026).
