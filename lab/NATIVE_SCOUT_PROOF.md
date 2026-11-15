# Native Codex-to-cloud research proof

The 15 November 2026 run completed the actual workflow: Codex called
`scout_research` through the stdio proxy; the bridge invoked the independently
configured AgentCore Harness; the cloud selected four local tool calls through
the HTTP proxy; and the local provider server made 101 GitHub reads and one
Notion read. The [canonical report](../results/scout/native-cloud-20261115/report.md)
contains three freshly checked contribution issues. It did not loosen filters
to fill the requested five slots.

The [offline oracle](../results/checks/native-scout-oracle.json) verifies 277
normalized events, three secret-credential fingerprints and one additional typed
AWS role principal. Provider-header observations establish direct use of GitHub
and Notion tokens. The SDK credential provider plus STS establish the temporary
AWS binding, labelled implicit. Delegated paths describe the operation's route;
they do not assert that Codex or the cloud received provider tokens.

## Explicit source links

1. The native Codex completed-item ID matches the original OTLP call ID. Its
   returned cloud session is preserved in the [redacted result excerpt](../captures/native-scout-20261115/codex-result-link.json).
2. That session identifies the bridge root, which records its incoming proxy
   session and JSON-RPC request fingerprint. The derived correlation record
   adds this independently verified parent to the proxy node. It is labelled
   as derived evidence, not an agent-emitted event.
3. All four cloud tool-use IDs match the original CloudWatch spans. Bridge
   records map each ID to its downstream HTTP request; provider records retain
   that request fingerprint at the authorization-header boundary.
4. All three W3C trace IDs sent by the bridge match native cloud traces. AWS
   inserted three ingress parents that are absent from this 44-span export.
   Their missing spans remain disclosed; direct parent equality is not claimed.

The [publication record](../captures/native-scout-20261115/publication.json)
lists original checksums and redactions. Cloud auth identifiers, account/resource
attributes, arguments and model content were removed. Native source IDs and
selected status fields remain. The inventory combines scanned provider identities
with the actual provider-server binding map and observed AWS runtime bindings;
temporary AWS credentials are not claimed to have been found in a static file.

## Results and limits

- [Credential lineage](../results/native-scout-20261115/lineage.txt): six aggregate
  rows, all credential paths resolved, zero quarantine records and zero binding
  findings against the reviewed inventory. Repeated native/proxy observations
  remain visible in the path; they are not extra provider reads.
- The original cloud run contains two nonfatal `RetrieveMemoryRecords` 403 spans.
  The missing action was subsequently scoped to the existing lab memory. A
  [fresh public follow-up](../results/checks/memory-live-followup.json) observed
  two successful retrieval spans and zero native error spans. Earlier errors
  were retained, not rewritten as successes.
- Local services were stopped with Ctrl-C, so their capture manifests are
  incomplete by design. The completed call/result pairs and report are checked
  independently. The first run failed before any recorded cloud invocation;
  its credential-helper issue is covered by a Windows stdin regression test.
- The report reader initially hit Windows' default text encoding; UTF-8
  revalidation passed on the existing capture without repeating model calls.

Recheck the published evidence without AWS access:

```powershell
python scripts/verify-native-scout.py --out private/new-native-verification.json
go test ./internal/adapters ./lab/cloud
```

Run a fresh real capture using [NATIVE_SCOUT_CAPTURE.md](NATIVE_SCOUT_CAPTURE.md).
Budget and retained-resource cleanup are documented in [RUNBOOK.md](RUNBOOK.md).
The evaluator's acceptance of AgentCore and the final human recording/rehearsal
remain external delivery steps.
