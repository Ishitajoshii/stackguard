# Capture the real research workflow

This Windows runner starts the provider server, HTTP proxy and metadata-only
collector; invokes the installed Codex CLI; and lets Codex call the stdio-wrapped
cloud bridge once. It requires an authenticated isolated Codex home, reviewed
skill summary, valid dedicated tokens, a READY Harness, active cloud telemetry
and an approved cost allowance. It does not provision AWS resources or copy
credentials.

```powershell
python scripts/native-scout-check.py --codex C:/path/to/codex.exe --codex-home private/your-login/codex-home --profile profiles/default.yaml --summary profiles/summary.json --bin-dir bin --out private/native-scout-new
```

The output directory must be fresh and under ignored `private/`. Full research,
CLI output and service diagnostics stay there for review. Only the provider
server receives GitHub/Notion tokens. Codex and the bridge receive neither token;
the bridge obtains its own temporary AWS role credentials through the configured
profile. The original Codex rollout remains in the operator's isolated home so
its call/result identifiers can be inspected without enabling prompt logging in
the metadata collector.

A successful runner result checks one completed native scout call, a complete
canonical research report and both provider credential fingerprints. This is
not sufficient by itself to establish native-cloud lineage. Export the actual
CloudWatch records, inspect their schema, and validate every cross-source link.
The bridge records the W3C trace parent sent on each SDK invocation; native cloud
spans must independently preserve that context before it is used as a join.
CLI item IDs and native Codex call IDs are different namespaces; never equate
them or correlate sources only because their timestamps are nearby.

The runner stops its local services with Ctrl-C. Those deliberate interruptions
produce incomplete capture manifests even when the completed request pairs are
valid; document both facts when publishing. It does not revoke provider tokens,
delete the cloud agent or remove tracing settings needed for a later recording.
