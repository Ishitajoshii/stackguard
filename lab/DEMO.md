# Five-minute live demo and handoff

This is a rehearsal script, not a recording. The [completed native research capture](NATIVE_SCOUT_PROOF.md) separately proves the cloud flow and its three rechecked issues.

Before spending on a fresh recording run, rehearse offline with `python scripts/verify-native-scout.py --out private/demo-verification.json` (choose a fresh output filename), then open `results/native-scout-20261115/lineage.txt`, `results/scout/native-cloud-20261115/report.md` and `SCALING.md`. These are saved evidence, not a new live run. For the live portion, use [the verified native runner](NATIVE_SCOUT_CAPTURE.md); keep its private raw outputs off screen. Explain the three missing AWS ingress-parent spans and the separately corrected optional memory reads rather than hiding them.

| Time | Show | Explain |
|---|---|---|
| 0:00–0:40 | Six-question profile and reviewed résumé skill summary | Inputs vary by person; raw PDF stays local. |
| 0:40–1:30 | Codex calls scout_research through the stdio proxy | The independent AWS agent selects actual inline tools. |
| 1:30–2:15 | A downstream HTTP tool call and checked shortlist | Provider tokens stay in the local tool server; report citations and uncertainty matter. |
| 2:15–3:15 | Scan inventory and trace --lineage | Full fingerprints, direct versus implicit observations, explicit request joins and unknown links. |
| 3:15–4:15 | One original direct/implicit/delegated chain and two real controlled findings | Compare native sources with supplementary bridge mappings. Do not substitute algorithm fixtures. |
| 4:15–5:00 | Scale table, Ctrl-C exit 17 and remaining gates | Explain what the measurements establish and what remains untested. |

Before recording: choose a fresh run folder; start the collector, tool server and proxy; verify the restricted AWS caller and harness are ready; use a terminal without secret values or private résumé text visible. Run the same commands from README in a clean checkout with credentials provided separately. Check that every visible link belongs to the intended account/profile.

For the explanation, walk through the code in this order: internal/scan (walker/worker/coordinator), internal/proxy (session-owned kestrelShard and transport pumps), internal/pipeline (bounded queues and external sort), internal/graph (disk lookup and explicit ancestry), internal/trace (stable reduction and risk evidence). Explain who closes each channel, why cancellation is selected on blocked sends, why timestamps cannot establish delegation, and why role IDs are not secret digests.

Record at most five minutes, review the recording for readability and private data, then add the actual recording reference. Do not upload or send the assignment automatically. Final destination, evaluator acceptance and the human explanation remain Ishita's steps.
