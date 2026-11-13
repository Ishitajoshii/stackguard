# StackGuard

A Go credential-inventory and trace-analysis CLI, with a customizable OSS contribution scout as its lab application. The core commands, both MCP proxy transports, local research workflow, cloud SDK bridge, profiling and benchmark runners are implemented. Live Codex → AgentCore acceptance remains unverified until dedicated credentials and cloud configuration are supplied.

## Build and check

Use Go **1.24+** (tested with 1.26.5), Git, and a C compiler on PATH for race tests. Python 3 with `pypdf` is needed only for local PDF extraction; Windows benchmark scripts use Python's standard library. AWS CLI and Codex CLI are needed only for the cloud lab. No EC2 is required.

```powershell
# Choose any disk with enough room; D: is this machine's choice.
$env:GOCACHE = 'D:\stackguard-data\go-cache'
$env:GOMODCACHE = 'D:\stackguard-data\go-mod'
$env:TEMP = 'D:\stackguard-data\temp'
$env:TMP = $env:TEMP
New-Item -ItemType Directory -Force $env:GOCACHE,$env:GOMODCACHE,$env:TEMP | Out-Null
go mod download
go build -o bin/ ./cmd/...
go test -race ./...
go vet ./...
python -m unittest discover -s lab -p 'test_*.py'
```

`bin/nhi-probe.exe`, `oss-scout.exe`, `sg-collect.exe` and `sg-bench.exe` are generated locally. Dependencies are pinned in go.mod/go.sum: TOML/YAML parsing, the official AWS SDK in the lab, and goleak for tests. Core scanning, proxying, sorting and graph reduction use the standard library; no agent orchestration framework is embedded in the graph. Go 1.24 is the AWS SDK's minimum. Verify these choices against the evaluator's permitted-library wording.

## Configure a different person

```powershell
.\bin\oss-scout.exe init --profile profiles/default.yaml
.\bin\oss-scout.exe prepare --profile profiles/default.yaml --summary profiles/summary.json
# Inspect summary.json and set its JSON field "reviewed" to true.
.\bin\oss-scout.exe doctor --profile profiles/default.yaml
.\bin\oss-scout.exe run --profile profiles/default.yaml --summary profiles/summary.json --yes --out results/scout/run-001
```

The six questions cover GitHub handle, résumé PDF (or skip), interests, languages, effort and a dedicated Notion page (or skip), followed by a correction/save step. `configure` edits an existing profile. Paths with spaces are supported. PDF parsing stays local, preserves page references and does not perform OCR. Only the reviewed skill summary is offered to the researcher; a detected mention is not a claim of proficiency.

Use [the editable profile](lab/examples/profile.yaml) for unattended setup. CLI flags override saved fields; allowlisted Notion preferences fill only missing saved/CLI preferences, then defaults apply. Advanced controls include `--hours`, `--activity-days`, `--candidates`, `--issues`, `--results`, `--max-queries`, `--exclude`, and `--types`. A Notion page may contain lines such as `interests: backend, databases` or `hours_budget: 8`. Arbitrary page instructions, linked pages, child pages, databases and synced content are not followed.

Credentials are supplied separately to the tool-server terminal through `STACKGUARD_GITHUB_TOKEN` and `STACKGUARD_NOTION_TOKEN`. Keep values out of command history; for example, PowerShell 7 can use `$env:STACKGUARD_GITHUB_TOKEN = Read-Host -MaskInput`. GitHub public reads can also work without a token, with lower provider limits. Doctor reports presence only, not validity. Notion access is restricted to the configured page subtree and still depends on its integration sharing permissions.

A report contains scored projects with open, unassigned, unlocked issues, license evidence, source links, uncertainty, effort ranges and first steps. It rechecks repository/issue/linked-PR state before selection, reports sparse matches honestly and never posts a comment. Scores are fixed heuristic weights (30/25/20/15/10), not an AI's confidence score. A linked open PR is conservatively excluded; unassigned does not mean nobody is working on it. API pagination and query limits are described in report coverage. Restart the server for a fresh bounded query budget.

## Capture the cloud lab after credentials are ready

Read [the runbook](lab/RUNBOOK.md) before a paid run. Choose an available region/model, an assumed caller role and a separate execution role. Set `aws_profile`, `aws_region`, and `harness_arn` in the personal YAML; no account, model or role is hard-coded. Root and long-term IAM-user callers are rejected by the bridge. Deployment input can be prepared offline:

```powershell
.\bin\oss-scout.exe cloud-plan --execution-role $ExecutionRoleArn --model $PermittedModel --out private/plan-001
# Explicit deployment, only after reviewing the plan and actual permissions:
.\scripts\deploy-harness.ps1 -InputFile private/plan-001/harness.json -Region $Region -Profile $AwsProfile -OutputFile private/harness-001.json
```

Check `Get-Help ./scripts/deploy-harness.ps1` for parameter names before running. No deployment occurs during build, doctor, tests or setup. The harness has seven named read-only inline tools, no wildcard built-ins, six iterations, 2,048 output tokens and a 60-second limit per invocation. The bridge additionally caps handoffs at eight and the complete workflow at five minutes. These are usage bounds, not a dollar cap.

Use separate terminals at the repository root (fresh capture directories each run):

```powershell
# Terminal 1, dedicated GitHub/Notion tokens only here:
.\bin\oss-scout.exe serve --profile profiles/default.yaml --summary profiles/summary.json --listen 127.0.0.1:8766 --out captures/run-001/tools
# Terminal 2:
.\bin\nhi-probe.exe proxy --mode http --listen 127.0.0.1:8765 --upstream http://127.0.0.1:8766/mcp --agent agentcore-researcher --client local-bridge --out captures/run-001/http
# Terminal 3, optional native telemetry collector:
.\bin\sg-collect.exe --listen 127.0.0.1:4318 --out captures/run-001/native-local
# Terminal 4, standalone cloud flow:
.\bin\oss-scout.exe cloud --profile profiles/default.yaml --summary profiles/summary.json --out captures/run-001/bridge
```

For Codex delegation, run `./scripts/configure-lab.ps1 -Profile profiles/default.yaml -Summary profiles/summary.json -RunName run-001` to generate an isolated configuration with your actual paths. This prepares files only. Alternatively, copy [the Codex template](lab/examples/codex.toml) to an isolated `CODEX_HOME`, replace its absolute paths and fresh run directory, and authenticate Codex using your own account. The template wraps `oss-scout bridge` in `nhi-probe proxy --mode stdio`. Ask Codex to call `scout_research`. Configure its OTLP/HTTP exporter with `protocol = "json"`; the collector rejects protobuf and discards prompt/output/account fields. An isolated home does not inherit your existing login. Never copy general credential stores into captures.

The cloud chooses actual inline function calls. The bridge sends their results back with matching tool-use IDs and the same AWS session. `github_shortlist` produces the independently checked report; an agent response without that result fails instead of publishing an unchecked shortlist. Model commentary is saved separately. Native cloud export and validation against real Codex/AgentCore records remain acceptance steps; bridge records are supplementary, not replacements.

## Probe commands

```powershell
.\bin\nhi-probe.exe scan --root D:/stackguard-data/repos --config codex=private/codex-home/config.toml --config agentcore=private/agentcore-export.json --agent codex --workers 4 --temp D:/stackguard-data/temp --out results/scan/run-001
.\bin\nhi-probe.exe trace --input proxy=captures/run-001/http/proxy.jsonl --input bridge=captures/run-001/bridge/SESSION/bridge-events.jsonl --input normalized=captures/run-001/tools/tool-events.jsonl --input codex-otlp=captures/run-001/native-local/native-otlp.jsonl --inventory results/scan/run-001/inventory.jsonl --lineage --workers 4 --temp D:/stackguard-data/temp --out results/trace/run-001
```

Replace `SESSION` with the generated cloud directory. Supported adapters: `normalized`, `proxy`, `bridge`, `otlp-json`, `codex-otlp`, `cloud-otlp`, `cloudwatch-export`. Supply only existing original inputs. A CloudWatch export must contain supported OTLP span envelopes or normalized records; arbitrary service log text is quarantined rather than guessed.

Scan emits candidate fingerprints and binding locations, not valid-key claims. It skips symlinks/special files, scans binary files as bytes and does not decode UTF-16. Configs/JSONL records are bounded to 1 MiB. Full SHA-256 identifiers drive deduplication; shortened masks are display-only. An AWS access/secret/session tuple is one credential, while a role ARN is a separate typed principal. Optional `--verify-owned private/owned.json` accepts an explicit fingerprint/provider/environment allowlist, currently for GitHub and Notion; it never validates credentials found in public repositories.

Trace emits `events.jsonl`, `edges.jsonl`, `findings.jsonl`, `lineage.txt`, `quarantine.jsonl` and a completion manifest. Direct use means presence at an observed call boundary; provider-header observations are stronger evidence than environment bindings. Implicit use is a runtime binding, not proof of API consumption. Delegated lineage follows explicit session/span/request links. No timestamp guessing or credential-transfer inference is used. Unknown, missing and ambiguous links remain visible. Calls count attempts, including failed ones; completion records do not inflate counts. Risks cover evidenced passthrough, use by at least three agents, unused-in-window candidates and uses absent from the supplied binding snapshot.

All output directories must be fresh. Exit codes: 0 successful operation, 2 failure/partial validation, 17 interrupted. Ctrl-C leaves whole persisted JSONL records and an incomplete manifest; forced termination or disk failure may prevent manifest publication. No stdout diagnostics enter stdio MCP traffic.

Scan writes redacted `observations.jsonl` as candidates arrive, before producing the sorted, deduplicated `inventory.jsonl`. Observation order can vary with workers and is not the canonical inventory. On interruption the observations remain useful even if final deduplication has not started. Interrupted scan/trace runs retain their temporary spill workspace and record its location in the manifest; remove that exact workspace after the process has stopped and the evidence has been reviewed. Large spill deletion is deliberately outside the interrupt path. Forced termination can still prevent a manifest, as documented in SCALING.md.

## Architecture and goroutines

Scan has one walker, N workers with 64 KiB read buffers, bounded queues and one output coordinator. The walker closes jobs; workers finish or see cancellation; a waiter closes results. The coordinator drains results and finishes evidence. Proxy routes session state to N `kestrelShard` goroutines; each owns its request map. Stdio uses separate input/output pumps and waits for the child. HTTP forwards JSON or SSE without rewriting protocol bodies and cancels upstream work when clients disconnect.

Use [the goroutine walkthrough](lab/GOROUTINES.md) to rehearse channel ownership, cancellation, EOF versus interrupt, and the limitations of OS-blocked operations. [The feasibility checkpoint](lab/FEASIBILITY.md) separates demonstrated local behavior from live setup gates.

Trace spills normalized events and graph nodes to disk, externally sorts 8 MiB chunks with eight-way merges, then resolves explicit ancestry through a disk index. Session-partitioned workers share an immutable index, not mutable graph maps. A serial reducer gives deterministic edge ordering. Queues block producers rather than grow without limit. Cancellation stops producers, unblocks sends, joins workers, closes children/listeners and writes incomplete provenance. Windows process-level HTTP shutdown is measured in [SCALING.md](SCALING.md); other acceptance gaps are listed there.

## Evidence and remaining gates

See [SCALING.md](SCALING.md) for measured workloads and raw files, [DOSSIER.md](DOSSIER.md) for signatures, sources and attribution limits, and [the demo guide](lab/DEMO.md) for the remaining live walkthrough. Tests use explicitly labelled algorithm fixtures; they do not fabricate agent activity.

The [six-page dossier PDF](output/pdf/stackguard-dossier-draft.pdf) is a visually checked research draft with live-evidence gaps labelled. Regenerate it with `python scripts/render-dossier.py` using the optional authoring dependencies in [the dependency inventory](lab/DEPENDENCIES.md). Recheck page count, links and rendered pages after inserting live evidence.

Still pending: evaluator acceptance of AgentCore instead of the named Bedrock Agents option; restricted live credentials/model access; original native local/cloud captures and hand-checked three-credential lineage; two real misconfiguration reproductions; the full 5 GB scan and 10-million-event target; final live demo recording and human rehearsal. Builds and tests do not close those gates. PRD.md and TODO.md are ignored local planning files; all reviewer-facing instructions are tracked here.

Codex generated and revised the implementation, configuration, docs and tests, and ran the recorded automated checks. No claim is made that Ishita has manually reviewed all code or completed the explanation/demo. Manual review should focus on the real original evidence and the goroutine/shutdown explanation above.
