# StackGuard nhi-probe

Go tooling to map agent credential use, with a customizable OSS contribution scout as its real lab workflow.

## Current implementation

The repository currently contains a tested Go CLI entry point, local PDF extraction helper and setup evidence. **scan, proxy, trace, the scout questionnaire and the cloud bridge are not implemented yet.** No real cloud-agent capture or scale result is claimed.

```powershell
go build ./cmd/nhi-probe
.\nhi-probe.exe --help
.\nhi-probe.exe --version
go test ./...
go test -race ./...
```

The initial CLI accepts help/version and rejects unavailable commands with exit 2. It does not echo unknown argument values into errors. Interrupt exit 17 is a requirement for the future processing pipeline and is not implemented in this scaffold.

## Local setup

Use native Windows with Go and a working C compiler for race tests. The current Go core uses only the standard library. See [toolchain evidence](lab/TOOLCHAIN.md), [race preflight](lab/RACE_PREFLIGHT.md) and [machine capacity](lab/ENVIRONMENT.md).

On this machine, use the selected D: location for large data and caches:

```powershell
$env:GOCACHE = 'D:\stackguard-data\go-cache'
```

This is a machine-specific example; other users can choose their own data/cache path. Do not put the 5 GB repository corpus on a nearly full drive.

For local PDF extraction, follow [PDF setup](lab/PDF_SETUP.md). Check mode prints counts only; explicit private JSON output preserves page references. The helper does not upload the resume or perform OCR.

## Planned real lab

Codex CLI will delegate through a transparent stdio proxy and a local bridge to an AWS AgentCore researcher. Cloud-selected inline function requests will return to the bridge, which will call real GitHub/Notion tools through a Streamable HTTP proxy. Native Codex and AWS telemetry plus explicit request mappings will support trace reconstruction.

The initial public example preferences are in [lab/examples/ishita.yaml](lab/examples/ishita.yaml). Personal paths, Notion page identifiers and credentials belong in ignored local configuration. See [private-data handling](lab/PRIVATE_DATA.md).

## Open gates

- Complete default-profile AWS sign-in and validate model/account access.
- Obtain and wire dedicated restricted GitHub and read-only Notion lab tokens.
- Prove cloud inline-function continuation and native local/cloud capture.
- Resolve evaluator acceptance of AgentCore in place of the assignment's named Bedrock Agents option.
- Implement and validate all scan/proxy/trace, scout and scale requirements.

The cloud budget is approximately US$5; follow [the runbook](lab/RUNBOOK.md). The PRD and TODO are local ignored planning documents. Reproduction instructions and final evidence must be available in tracked files without them.

## AI assistance

Codex was used to turn the supplied assignment and user preferences into planning documents, create repository setup and environment evidence, write the initial CLI and PDF helper, and execute the recorded checks. This section must be updated with subsequent implementation and review work. Generated code has only the validation explicitly recorded here; a coding-agent run is not evidence of completed cloud or benchmark gates.
