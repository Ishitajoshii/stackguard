# Original Codex file-tool evidence

On 14 November 2026 (IST), the installed Codex CLI 0.166.0 used the operator's existing ChatGPT sign-in to call the real `repository_file` MCP tool through both proxy transports. Each call read 1,454 bytes from the Go checkout's README and returned SHA-256 `8247b7c5de1e74854aac1a08aa5894444d1d33b4045c70d5cc3367ad0e25c3f3`, independently checked against that file. No GitHub/Notion token or AWS call was involved.

| Transport | Original metadata-only capture | Normalized result |
|---|---|---|
| Stdio | [Capture and summary](../captures/codex-file-stdio-20261114) | [Four events, complete trace](../results/native-codex-file-stdio) |
| Streamable HTTP | [Capture and summary](../captures/codex-file-http-20261114) | [Four events, complete trace](../results/native-codex-file-http) |

Each proxy captured one call and one successful completion. Native Codex emitted the matching tool name/server, success, timestamp and its own conversation/call IDs, plus a separate `functions.exec` wrapper result. Each combined trace therefore has four source observations, not four independent repository reads. Credentials, edges and lineage are empty for this workload. Native call IDs and proxy JSON-RPC request IDs differ; this evidence does not establish an explicit cross-source parent mapping. No join is fabricated from timestamps or the fact that only one MCP call occurred.

The HTTP proxy was stopped by actual Ctrl-C, exited 17 in 0.054 s and published a valid incomplete manifest. The collector also published a valid incomplete manifest on deliberate Ctrl-C. Codex's managed stdio child exited without a final proxy manifest; its two complete JSONL records remain available, and that missing manifest is reported in the summary. A completed Codex turn is not presented as a graceful proxy-shutdown test. The first private attempt was blocked by Codex's noninteractive approval policy and did not reach the proxy; it is not a passing run.

Prompts and agent-response logging were disabled. The collector discarded bodies, arguments, output values and account fields before persistence. Full CLI diagnostics stay under ignored `private/`; only reviewed native metadata, proxy observations and the checked file-hash summary are tracked. A [real one-record fixture](../internal/adapters/testdata/codex-file-tool.otlp.json) is cut from the original stdio capture; its [provenance](../internal/adapters/testdata/codex-file-tool.provenance.json) records the source checksum and line. The native adapter test verifies these original identifiers, timestamp and outcome without inventing a credential or parent.

Reproduce on Windows with an existing Codex sign-in and built binaries:

```powershell
python scripts/native-codex-check.py --codex C:/path/to/codex.exe --repo D:/path/to/real/go-checkout --out private/native-stdio-new --transport stdio
python scripts/native-codex-check.py --codex C:/path/to/codex.exe --repo D:/path/to/real/go-checkout --out private/native-http-new --transport http
```

The harness uses a read-only sandbox, ignores user MCP configuration, enables only the file tool on its test server, and scopes explicit approval to that tool. It makes no persistent Codex configuration changes or credential copies. The operator must already have authorized this local read-only test. Other installed plugins can still appear in Codex's catalog; the harness verifies the completed MCP call used the intended server. Configuration references: [MCP tool approval](https://learn.chatgpt.com/docs/extend/mcp) and [native telemetry](https://learn.chatgpt.com/docs/config-file/config-advanced).

Cloud telemetry, the real research workflow, three owned lab credentials, credential-bearing native lineage and controlled misconfiguration reproductions remain pending.
