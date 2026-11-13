# Dependency boundaries and review status

Reviewed against the [original assignment](https://docs.google.com/document/d/1QUQ0Xk16P98GToYWu1BNoJoiKbfRkT_4YEpa2BM85lA/edit) on 14 November 2026. Exact versions and transitive Go checksums are pinned in go.mod/go.sum. Its allowed Go dependency categories include YAML/TOML parsers, goleak and official cloud SDKs; the dependencies below fit those categories. The probe's external link dependencies were checked with `go list -deps`. AgentCore's acceptance as the cloud agent remains a separate open decision.

| Layer | External dependencies | Purpose |
|---|---|---|
| Go probe executable | github.com/pelletier/go-toml/v2 v2.4.3 | Parse explicitly supplied Codex TOML configuration. All scan, detection, transport, sorting, graph, output and verification code otherwise uses the standard library. |
| Go tests | go.uber.org/goleak v1.3.0 | Detect leaked pipeline/proxy goroutines; not linked into the shipped executable. Go's race detector uses the installed C toolchain. |
| Scout profile | gopkg.in/yaml.v3 v3.0.1 | Human-editable user profiles, isolated to lab/scout. |
| AWS researcher bridge | aws-sdk-go-v2 v1.48.0, config v1.33.18, bedrockagentcore v1.58.0, sts v1.52.0 | Official credential provider, signing, STS identity and Harness event-stream client. Transitives in go.mod include credentials, SSO/sign-in, IMDS, Smithy and eventstream. Not imported by nhi-probe. |
| Native telemetry collector | Go standard library | Local HTTP JSON collector and field allowlist; no third-party collector process required for the supported JSON route. |
| Local PDF extraction | Python and pypdf 6.21.3 | Local résumé text extraction with page references. Pinned in lab/requirements-pdf.txt; no OCR or upload. |
| Optional dossier authoring | Python, reportlab (observed 5.1.0), pypdf and Poppler | Regenerate, inspect and render the draft PDF. Not required to build/run the Go probe or scout. |
| Windows measurement scripts | Python standard library, ctypes Win32 APIs | Process control, OS peak working set/commit, real request timings and output hashes. |
| Setup/operator tools | Git, GitHub CLI, Codex CLI, AWS CLI, PowerShell | Public clone/private repository operations, local agent and explicit cloud setup. Versions are recorded in TOOLCHAIN.md. |

No gitleaks/trufflehog wrapper, general secret-scanning library, third-party graph engine or third-party MCP SDK is used. The core implements the pinned transport subset directly. Source layout separates cmd/nhi-probe and internal from the secondary cmd/oss-scout, cmd/sg-bench, cmd/sg-collect and lab packages. The module's Go 1.24 minimum is imposed by the pinned AWS lab SDK; local validation uses Go 1.26.5 and CI follows go.mod.

To inspect the actual probe link dependencies:

```powershell
go list -deps -f '{{if not .Standard}}{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}{{end}}' ./cmd/nhi-probe | Sort-Object -Unique
go list -m all
```

The first command was checked: its external result is only go-toml/v2. The repository module itself also appears. Python PDF tools and Windows measurement helpers are separate authoring/lab utilities, not Go probe dependencies. The assignment requires Go 1.22 or newer, so the module's Go 1.24 minimum satisfies that lower bound. This review covers dependencies and source layout; it does not establish acceptance of AgentCore, real-agent correctness, or completion of the scale gates.
