# Two controlled findings with real credentials

This experiment used the owner's actual GitHub and Notion lab tokens and one
successful read of `golang/go` through the real MCP tool server. It made no provider
writes and generated no synthetic events. It does not claim an agent delegation.

The [original provider-boundary events](../results/controlled-risks/original-tool-events.jsonl)
contain the GitHub token's full SHA-256 fingerprint, direct attribution, and the
matching successful completion. The private HTTP result was checked for success.
The server was stopped with Ctrl-C, so its [manifest](../results/controlled-risks/tool-manifest.json)
correctly says incomplete. The independently checked two-record call is narrower
evidence than a complete continuous service capture.

1. **Unused within the window — low severity.** The scanner read an actual local
   configuration containing both owned tokens. Only GitHub was used during the
   controlled window. The [full inventory](../results/controlled-risks/inventory-full.jsonl)
   and [finding](../results/controlled-risks/unused-findings.jsonl) identify the
   Notion token as unobserved in that window. This is not proof it is globally
   unused: the separate authenticated scout run did use Notion.
2. **Observed use missing from the snapshot — medium severity.** A second actual
   configuration snapshot deliberately omitted the GitHub token while the same
   real outbound call used it. The [reduced inventory](../results/controlled-risks/inventory-missing-github.jsonl)
   produces [the missing-binding finding](../results/controlled-risks/missing-binding-findings.jsonl).
   This reproduces incomplete credential wiring evidence, not unauthorized access.

These are two inventory conditions over the same original request, not two
independent agent sessions. The [verification](../results/controlled-risks/summary.json)
records that scope. Publication removed token masks and machine-specific inventory
paths, preserving fingerprints and binding agents; [source hashes and transformations](../results/controlled-risks/publication.json)
are recorded. The original event bytes are unchanged. Temporary plaintext scan
inputs were deleted by the runner's `finally` block.

Reproduce from the published evidence:

```powershell
bin/nhi-probe.exe trace --input normalized=results/controlled-risks/original-tool-events.jsonl --inventory results/controlled-risks/inventory-full.jsonl --out output/review-unused --lineage
bin/nhi-probe.exe trace --input normalized=results/controlled-risks/original-tool-events.jsonl --inventory results/controlled-risks/inventory-missing-github.jsonl --out output/review-missing-binding --lineage
```

Use fresh output paths. To repeat the provider call with your own credentials,
run `scripts/controlled-risk-check.py` with a private output directory, your
profile and reviewed summary. It reads only the two dedicated `.env` entries and
checks the exact expected fingerprints before accepting either finding.
