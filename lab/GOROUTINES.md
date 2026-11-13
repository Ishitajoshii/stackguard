# Explaining the concurrency

Use this alongside the code during rehearsal. It describes the implementation; it does not claim that the human explanation or live demo has happened.

## Scan: bounded work, one coordinator

In [scan.go](../internal/scan/scan.go), one walker discovers selected files and sends paths to `jobs`. N workers read files with their own 64 KiB buffer and report matches/byte counts to `results`. Both queues hold at most 2N entries. The main goroutine owns the spool and manifest counters, so workers never modify those counters concurrently.

The walker is the only sender that closes `jobs`. A waiter waits for the walker and workers, then closes `results`. The coordinator drains results until that close. A send uses `select` with `ctx.Done()` so a worker can leave if the coordinator has encountered an output error. Receivers do not close channels owned by senders.

After scanning, external sorting groups fingerprints for deterministic inventory output. The current per-identity binding limit is 4,096: exceeding it fails the run explicitly. It does not silently truncate or prove that arbitrarily large binding aggregates are supported.

The coordinator also writes redacted observations immediately. This separate, potentially duplicated stream survives interruption before deduplication; its ordering is not promised to match across workers. Canonical comparisons use the final inventory.

## Proxy: one owner for each session's pending calls

In [observer.go](../internal/proxy/observer.go), a stable hash assigns a session to one observer queue. Each observer goroutine exclusively owns its pending-call map. That avoids a shared map lock across sessions. Each queue has eight slots; each observation also has a one-slot acknowledgement channel. The transport waits for that acknowledgement before continuing, so telemetry pressure is visible instead of silently dropping records.

Both queue sends and acknowledgement waits can be cancelled. The acknowledgement channel is buffered so the observer can finish even if the waiting caller has left. These queues are not closed: the owner context stops their workers, and `Wait()` joins them after cancellation. Pending calls are removed on completion and capped at 4,096 per shard; crossing the cap produces an error.

The stdio proxy has two pumps: client input to child stdin, and child stdout to client output. A child exit or interrupt closes the relevant pipes and an `io.Closer` output, unblocking idle reads or output backpressure. The proxy joins both pumps and reaps the child. A child that closes stdout but stays alive gets a bounded grace period before cancellation. Tests cover this failure mode rather than assuming EOF means process exit.

HTTP handlers use request contexts. A disconnect cancels upstream work; observer cleanup uses the observer's still-live context to remove the pending request. A truncated response generates failed-completion metadata internally without changing the bytes already forwarded to the client.

## Lab stdio server: cancellation can pass a running tool

In [stdio.go](mcp/stdio.go), the input loop keeps reading control messages while at most eight tool goroutines run. Each call has its own cancellation function in a mutex-protected map. `notifications/cancelled` cancels the matching call; a separate write mutex prevents JSON responses from interleaving. A valid EOF waits for admitted calls to finish, whereas an interrupt closes closable streams and cancels work. The production tools must honour their contexts; Go cannot safely kill an arbitrary goroutine that ignores cancellation.

## Trace: the graph stays on disk

In [transform.go](../internal/pipeline/transform.go), a producer hashes the session key and sends each record to one of N queues, each with capacity two. Workers return results through an N-slot queue. The producer closes the worker queues, a waiter closes results, and the caller owns the output writer. A first error cancels the pipeline while the caller drains its remaining results so workers can finish.

Many records from one session route to one worker. This preserves per-session ownership but can reduce parallelism; increasing `--workers` does not guarantee a speedup. Crucially, that worker does not retain the session's full history. The immutable graph index, event spool and sort runs stay on disk. External sort uses 8 MiB chunks and an eight-way merge. Stable sorting after parallel work provides deterministic final output.

The bounds come from data structures, not just a Go GC memory target:

| State | Bound or storage strategy |
|---|---|
| Input record | `pipeline.Lines` rejects records exceeding 1 MiB. Normalized events allow at most 64 credential observations and 64 evidence references. |
| Parent history | `graph.Index` writes all nodes and request aliases to disk. Late parents are available before path resolution; there is no growing in-memory pending-parent map. |
| One ancestry walk | At most 128 node visits and a local cycle-detection set. Longer or cyclic paths carry an explicit unresolved reason. |
| Graph lookup | `pipeline.Table` stores both its fixed-width index and values on disk; binary search reads individual records with `ReadAt`. |
| Session count | Session IDs select a fixed worker queue. Trace does not allocate a history map per session. |
| Sort metadata | An on-disk run list and groups of at most eight merge readers, rather than one in-memory filename per run. |
| Credential aggregates | Sorted streams reduce one edge/identity group at a time. Evidence is capped at 64 references; complete per-event evidence remains in `events.jsonl`. Risk agent samples are capped at 64. |

These limits explain the design. Actual peak working set and commit measurements for the one-session and many-session load shapes are separate scale gates; a bounded design alone does not establish the 512 MB result.

## What Ctrl-C guarantees have actually been checked

CPU profiling now has one owner goroutine. It stops and finalizes each 30-second segment before starting the next, writes the completed filename to a streaming JSONL index, and selects cancellation directly rather than waiting for all data cleanup first. The caller closes its stop channel once and waits for the owner to close `done`; errors are read only after that synchronization. This bounds retained sampling metadata by a window instead of the duration of the whole benchmark. Rotation has small unsampled gaps. `nhi-probe` also lowers Go's soft memory target to 256 MiB while respecting an already stricter `GOMEMLIMIT`; disk and queue bounds still provide the structural limits, and OS memory measurements still determine pass/fail.

The CLI creates a signal-aware context. Cancellation stops intake and blocked channel operations, closes network/child resources and writes an incomplete manifest. Exit 17 distinguishes interruption from an ordinary failure. JSONL writes roll back short writes to the last whole-record offset; output close calls `Sync`, and completion status includes output flush failures. A forced kill or power loss can prevent manifest publication. These are not per-record power-loss durability guarantees.

An earlier large trace failed the two-second interruption check while deleting multi-gigabyte spill files before publishing its manifest. Interrupted runs now retain their spill workspace, record its location and leave deletion for an explicit later cleanup. Successful runs still remove their workspace. The correction is covered by retention/cancellation tests; early process-level interruption alone is not proof of every possible late-stage disk condition.

The measured process tests and limits are linked from [SCALING.md](../SCALING.md). The latest 1,000-in-flight transport run shut down in 0.115 seconds. Unit tests cover cancellation, idle input, blocked output and child shutdown; race checks and goleak address different failure classes. A filesystem operation stuck inside the OS may not respond to a Go context immediately, so measured two-second shutdown results apply to the tested local workloads.

The later [large-spill interruption](../results/bench/late-spill-interruption/summary.json) also passed: Ctrl-C returned 17 in 0.486 seconds after 9,930,978 events had been published. Every retained JSONL record parsed, the incomplete manifest matched that count, and the spill workspace was explicitly cleaned only after exit and validation. This closes the tested late-cleanup regression without promising arbitrary OS disk-stall latency.

A subsequent [long-profile run](../results/bench/trace-10m-memory-failure/interruption.json) missed the deadline after its incomplete data manifest had been written; the CPU profile was still incomplete when the harness terminated the process. Windowed profiling passes rotation/cancellation/race/leak unit tests and a [100k-event process check](../results/bench/profile-rotation-cancellation/summary.json): exit 17 in 0.337 s after two complete profile windows, with all partial events and finalized profiles valid. A [completed corrected-build 10M run](../results/bench/trace-10m-bounded-profiles-w4/summary.json) also stayed below the memory limit and finalized all 151 CPU windows. Successful completion does not establish large late-interruption latency; that regression remains running separately.

For rehearsal, trace one path aloud: Ctrl-C cancels the CLI context; the producer stops; worker sends choose cancellation; the waiter joins workers; the coordinator drains and closes output; the manifest says incomplete; the process returns 17. Then explain why closing a channel from an arbitrary receiver could panic another sender, and why explicit ownership avoids that race.
