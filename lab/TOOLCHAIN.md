# Toolchain preflight

Observed 13 November 2026 on native Windows amd64:

| Tool | Installed version |
|---|---|
| Git | 2.48.1.windows.1 |
| Go | 1.26.5 |
| Codex CLI | 0.166.0 |
| AWS CLI | 2.38.5 |
| GitHub CLI | 2.92.0 |
| Node.js | 24.16.0 |
| GCC | 15.2.0, MSYS2 UCRT64 |
| Python | 3.14 installation present |

Go reports `CGO_ENABLED=1`, `GOOS=windows`, `GOARCH=amd64`. These settings and GCC availability are prerequisites, not evidence that the race detector has passed. Run the race preflight separately.

AWS's configured default region is ap-south-1. The initial credential check reported an expired session; region configuration is not evidence of model access. No credentials or account identifiers are included here.

`pdftotext` was not found on PATH. Python's pypdf package is already installed; the PDF extractor is handled as a separate lab dependency rather than a Go core dependency.

Recheck versions with `git --version`, `go version`, `codex --version`, `aws --version`, `gh --version`, `node --version`, `gcc --version`, and `go env CGO_ENABLED GOOS GOARCH`.
