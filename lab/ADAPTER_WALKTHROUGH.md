# Adding an adapter from an original capture

The completed example is `Codex.Decode` in [native.go](../internal/adapters/native.go), registered as `codex-otlp`. Its [one-record fixture](../internal/adapters/testdata/codex-file-tool.otlp.json) is an unchanged `codex.tool_result` record selected from line 4 of the original [stdio native capture](../captures/codex-file-stdio-20261114/native/native-otlp.jsonl). The provenance file records the source checksum. Tests verify both the extraction and the normalized identifiers; this is separate from algorithm-only malformed/private-field tests.

1. Inspect an original source envelope and record its capture location, SHA-256 and selection. Preserve source identifiers and timestamps. Redact private data at collection, before checking anything in.
2. Implement `TraceAdapter.Decode([]byte) ([]model.Event, error)` in an adapter file/package. Decode the source envelope without changing graph, pipeline or risk code. Register the source name in `DefaultRegistry`.
3. Map only supported fields. Here, `conversation.id` is the session; `call_id` is the span/call ID; `event.timestamp` is the event time; `tool_name` is the target; and `success` supplies the outcome. The stable event ID hashes session, call and the native result sequence. Other Codex event names are ignored.
4. Validate each normalized event. Missing required IDs or invalid timestamps are errors. Native tool results identify an observed attempt, including failed attempts; the proxy's separate completion record does not add another attempt.
5. Leave unsupported credential and parent fields empty. This native record does not establish the proxy request ID, AWS role, credential transfer or cloud parent. Similar timestamps and tool names cannot supply those links.
6. Add a regression test against the real selected record, plus clearly labelled parser tests for invalid inputs. Verify exact native session, call, target, time and status, and verify that unsupported credentials/parents stay absent. Keep the raw capture checksum test so later fixture edits cannot silently manufacture evidence.

Run the example without cloud credentials:

```powershell
go test -race ./internal/adapters ./internal/trace
go build -o bin/walkthrough/nhi-probe.exe ./cmd/nhi-probe
./bin/walkthrough/nhi-probe.exe trace --input codex-otlp=captures/codex-file-stdio-20261114/native/native-otlp.jsonl --out results/local-adapter-walkthrough --temp D:/stackguard-data/temp --lineage
```

Choose a fresh output directory. The CLI reads one envelope per JSONL line; the pretty-printed unit fixture is passed directly to `Decode` only in tests. Expected CLI output from this original capture: two native events (the repository tool and its `functions.exec` wrapper), no credential edges, no invented lineage and a complete manifest. The complete native-plus-proxy examples are documented in [NATIVE_LOCAL_PROOF.md](NATIVE_LOCAL_PROOF.md). A new cloud source still needs its own real export and mapping review; supporting a generic OTLP envelope does not prove every provider's schema.

Adapters return normalized events through the same small interface, so the core graph does not depend on source-specific OTLP attribute names.
