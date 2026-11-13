# Windows race detector preflight

On 13 November 2026, the initial CLI tests passed with the installed Go 1.26.5 and GCC 15.2.0 toolchain on Windows amd64:

```powershell
$env:GOCACHE = 'D:\stackguard-data\go-cache'
go test -race ./...
```

Observed result: `ok github.com/Ishitajoshii/stackguard/cmd/nhi-probe` (2.265 seconds as reported by go test). The ordinary `go test ./...` run also passed, and `go build ./cmd/nhi-probe` produced a working `--version` response.

This proves the local race-detector toolchain can execute the current tests. It does not prove race freedom, leak freedom, cancellation behavior or concurrency correctness for the unimplemented scan/proxy/trace pipeline. Repeat the relevant checks after that implementation exists.

Keep build caches on D: while C: is short of space; this command changes the cache only for the current shell process.
