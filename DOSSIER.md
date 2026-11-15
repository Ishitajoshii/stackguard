# Credential research dossier

StackGuard inventories credential candidates and reconstructs observed use by agents. This dossier separates documented behavior, implementation contracts and measurements. Original evidence includes native Codex-to-AgentCore research through both proxies, three credential fingerprints, a checked shortlist, two controlled inventory findings and local-file load tests. The [native proof](lab/NATIVE_SCOUT_PROOF.md) records explicit source links and remaining trace gaps. Updated **15 November 2026**. The [source index](lab/SOURCES.md) maps claims to primary references and implementation evidence.

## 1. Credential stores and propagation

Codex MCP configuration can contain subprocess commands/arguments, environment values/references, remote URLs and authentication-header references. A configured reference establishes intended availability, not that a provider accepted a credential. StackGuard reads only explicitly supplied config files and resolves only their named environment references. Missing references remain unresolved identities. User authentication storage, OS credential stores and remote OAuth sessions are outside the scanner's coverage unless an explicit export is supplied. Codex telemetry must be enabled separately; its exporter and prompt controls are documented in the [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).

The lab uses an isolated Codex configuration, a local stdio bridge, an AWS Harness session and local read-only GitHub/Notion MCP tools. Subprocess environment inheritance can make a token available to a child without putting it into a JSON-RPC argument. Remote MCP credentials can instead appear in HTTP authorization headers. OAuth adds access/refresh tokens and a separate authorization flow; inspecting a server URL alone cannot discover those credentials. The [MCP transport specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports) describes the transport boundary; it does not make every tool call evidence of credential consumption.

The AWS SDK signs requests using its configured provider chain. Temporary access-key ID, secret access key and session token are one credential set. StackGuard hashes a length-delimited tuple so ambiguous concatenation cannot merge different sets. An access-key ID found alone is labelled as an identifier, not a complete usable credential. A role/session ARN is a typed principal and has no secret value to hash. [AWS temporary credential documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_request.html) explains the set; STS identity establishes the caller, not permission to invoke a model.

The bridge requires an assumed caller role, while Harness has its own execution role. Scoped permissions, model access and a [public inline-tool continuation](lab/CLOUD_HANDOFF_PROOF.md) now pass live checks. An explicit seven-tool allowlist uses `@name` inline selectors and excludes built-in shell tools; IAM separately restricts direct runtime-command APIs. Returning a cloud tool result uses the pending tool-use ID and the same session. Neither prompt instructions nor tool allowlists replace IAM. See [Harness security](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-security.html).

## 2. Three additional platforms: documented, not observed

| Platform | Relevant locations and authentication | Coverage limit |
|---|---|---|
| Claude Code | Project `.mcp.json`; user/local MCP scopes in `~/.claude.json`; environment expansion, remote headers and OAuth are documented. | Settings files and MCP state have distinct scopes. No Claude native capture or credential-store extraction was performed. |
| Cursor | Project `.cursor/mcp.json` and user `~/.cursor/mcp.json`; environment interpolation in supported fields, stdio environment files and remote authentication. | CLI/IDE behavior and OAuth stores require a real versioned capture before claiming parity. |
| GitHub Copilot CLI | User `~/.copilot/mcp-config.json`, with `COPILOT_HOME` customization; project MCP configuration has its own precedence. | VS Code's `.vscode/mcp.json` is a different config surface, not interchangeable evidence of CLI configuration. |

These findings come from the official [Claude MCP documentation](https://code.claude.com/docs/en/mcp), [Cursor MCP documentation](https://cursor.com/docs/mcp), and [Copilot CLI setup](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers). StackGuard does not advertise tested native adapters for these three additional platforms.

## 3. Eight provider detectors and verification limits

Patterns below describe this implementation's bounded heuristics, not guarantees about every current or future issuer format. The full expressions are in internal/secrets. Test values are algorithm fixtures and are never presented as real issued tokens. Public-corpus matches are not sent to providers. [GitHub's supported-secret-pattern reference](https://docs.github.com/en/code-security/reference/secret-security/supported-secret-scanning-patterns) also distinguishes credential types and availability of validity checks.

| Provider | Implemented signature/context | False-positive or missed-format risk | Owned-only read operation |
|---|---|---|---|
| GitHub | `gh[pousr]_` plus 20–255 alphanumerics, or `github_pat_` plus 20–255 alphanumerics/underscores | Samples can match; app/installation/PAT types differ. | `/user` for an appropriate user token; installation tokens need an installation-specific operation. |
| Notion | `ntn_` or legacy `secret_` plus 20–255 alphanumerics; named fields | Literal sample strings and evolving token formats. | `GET /v1/users/me`, then separately verify access to the configured page. |
| OpenAI | `sk-`, optional `proj-`/`svcacct-`, bounded alphanumeric/underscore/hyphen suffix; excludes `sk-ant-` | Prefix is not issuer proof; project restrictions affect accessible resources. | List models with the owned API key; API-key authentication differs from a ChatGPT login. |
| AWS | `AKIA`/`ASIA` plus 16 uppercase alphanumerics; explicit AWS environment tuple | An ID alone is not a secret; arbitrary secret/session strings need context. | Signed STS `GetCallerIdentity` on a complete owned session. |
| Slack | `xox[baprs]-` and bounded suffix | Multiple Slack token classes; strings in docs can match. | `auth.test`; inspect its `ok` result, not HTTP 200 alone. |
| Anthropic | `sk-ant-` plus bounded suffix; named field | Format changes and test strings; workspace policy may restrict use. | A documented read-only models operation with required authentication/version headers. |
| Google | `AIza` plus 35 alphanumeric/underscore/hyphen characters; named field | API keys differ from service-account credentials and can be API/IP/referrer restricted. | A minimal authorized read against the intended enabled service; there is no universal key-validity endpoint. |
| Azure | Explicit Azure-labelled key/token assignment or sensitive field; no universal prefix | Arbitrary strings and service-specific key formats; managed identity is not a static key. | A read against the intended resource endpoint with its documented authentication, if that service supports it. |

Authentication references: [Notion](https://developers.notion.com/reference/authentication), [OpenAI](https://platform.openai.com/docs/api-reference/authentication), [Anthropic](https://platform.claude.com/docs/en/api/overview), [Slack auth.test](https://docs.slack.dev/reference/methods/auth.test/), [Google API-key practices](https://docs.cloud.google.com/docs/authentication/api-keys-best-practices), and [Azure OpenAI authentication](https://learn.microsoft.com/azure/api-management/api-management-authenticate-authorize-azure-openai). The implemented optional verifier supports only GitHub and Notion and requires exact fingerprint/environment-value agreement. A rejection, permission failure and incomplete check are different outcomes.

## 4. What attribution evidence can establish

A proxy call record establishes the observed request, session and supplied credential presence. A completion record establishes the corresponding response status; it does not increment call count. A child environment binding is **implicit** evidence: a credential was available to that process. The tool server's outbound authorization-header hook provides **direct** evidence that a credential was attached at a provider request boundary. Even that does not prove authorization succeeded unless the response supports it. The observer stores digests, never full headers or request bodies.

Delegated lineage requires explicit parent/span/request mappings. The bridge maps the inbound MCP request to the cloud session and cloud-selected tool-use ID, then to the downstream MCP request. These are supplementary application observations. Native Codex tool events can supply conversation/call identifiers; native cloud OTLP spans can supply trace/span ancestry. A label such as `agentcore-researcher` is a configured assertion until native evidence supports the identity. Timestamp proximity cannot supply a missing parent or prove token transfer.

Adapters validate required fields, quarantine unmappable records and preserve checksums. Disk indexing resolves late parents; paths retain agents and tool targets. Duplicate source IDs must agree. The [native oracle](results/checks/native-scout-oracle.json) validates 277 events, the flat CloudWatch span schema, the Codex result/session link and four cloud tool-use IDs. Three AWS ingress-parent spans are absent; no substitute parents are invented. Native auth attributes and payloads are removed before publication. Bounded aggregates retain up to 64 evidence references; individual references remain in normalized events.

No observed use within a supplied capture window is not proof that a token is permanently unused. Missing inventory bindings are not proof of unauthorized use. Capture completeness, source selection and the observation interval must accompany each conclusion. Local-file measurements test transport, replay scale, bounded memory and cancellation; they contain no provider-secret attribution oracle.

## 5. Six misconfigurations and controlled reproductions

| Misconfiguration | Severity and reasoning | Evidence/remediation |
|---|---|---|
| Credential passthrough across agents | High: expands the principals with access to a reusable secret. Delegation alone is insufficient to flag it. | Detector requires an explicit transfer observation and direct presence. Delegate operations, keeping credentials at the tool boundary. Real positive capture pending. |
| One credential used by at least three agents | Medium: weakens isolation and attribution; shared service identities may be intentional. | Detector groups full identity and distinct agent labels. Use separately scoped identities or document justified sharing. Algorithm test only so far. |
| Unused credential in the observation window | Low: may be unnecessary exposure, but incomplete captures can explain absence. | [Owned-token reproduction](lab/CONTROLLED_RISKS.md): Notion is inventoried but unobserved during a real GitHub-only read. Confirm coverage before revocation. |
| Observed use absent from binding snapshot | Medium: config drift or missing evidence; not automatically unauthorized access. | [Owned-token reproduction](lab/CONTROLLED_RISKS.md): the same real GitHub read is absent from a deliberately reduced inventory. Reconcile snapshot scope and runtime bindings. |
| Overbroad provider/IAM scope | High when unrelated data or command execution becomes available. | Restrict GitHub reads, Notion page sharing, AWS role/model resources and direct runtime APIs. Tool allowlists complement IAM; they do not replace it. |
| Plaintext credential storage in a shared/tracked file | High when another principal can read or receive it. | Keep dedicated values in private environment/credential stores, restrict access, rotate exposed values. Planned ignored lab-only reproduction pending. |

The [two controlled demonstrations](lab/CONTROLLED_RISKS.md) use owned credentials and one original authenticated GitHub read. A full inventory produces a qualified unused-Notion finding; a deliberately reduced snapshot produces a missing-GitHub-binding finding. These are two inventory conditions over the same request, not two independent agent sessions or a credential-transfer demonstration. Published artifacts retain source checksums, fingerprints, capture boundaries and disclosed transformations; temporary plaintext scan inputs were removed.

## 6. Evidence status and delivery limit

The complete [native research run](lab/NATIVE_SCOUT_PROOF.md) returned three rechecked issues after 101 GitHub reads, one Notion read and three AWS invocations. The oracle validates three secret-credential fingerprints plus a separately typed role, explicit source links and complete credential paths. Two nonfatal memory-read errors remain in the original capture; a scoped correction passed a fresh native follow-up with zero errors. [Two controlled inventory findings](lab/CONTROLLED_RISKS.md) have separate owned-token evidence.

| Evidence | Available artifact | What remains to establish |
|---|---|---|
| Real MCP calls | [Native research](lab/NATIVE_SCOUT_PROOF.md) exercises both transports; [load test](results/bench/transport-w4-r5/summary.json) reaches 1,000 simultaneous handlers. | Human recording/rehearsal; duplicate source observations stay visible. |
| Credential candidates | [Complete 5.84 GB scan](results/bench/scan-corpus-full-w16/summary.json); [three-credential oracle](results/checks/native-scout-oracle.json). | Public-corpus candidates are never verified against providers; temporary AWS bindings are observed at runtime. |
| Trace load | [10-million-event replay](results/bench/trace-10m-bounded-profiles-w4/summary.json), extreme session shapes and [worker results](SCALING.md). | Paused/shared-disk timings do not establish worker speedups. Scale replay and native correctness are separate evidence. |
| Automated correctness | Race/leak/fuzz checks, cancellation measurements and original native-schema fixtures. | Three cloud ingress-parent spans are absent; recording and evaluator acceptance remain external. |

The [cost record](results/checks/cloud-cost-final-estimate.json) includes all attempts, unknown-usage reserves, runtime assumptions and the retained KMS key. Its conservative scenario is below USD 5; same-day billing remains estimated. [Cleanup](lab/RUNBOOK.md) follows the recording, with exact resource receipts preserved privately. No further paid test is scheduled.

The repository contains the reviewed source captures, oracle, results and research. The required demo recording and evaluator acceptance of AgentCore in place of the assignment's named cloud option remain outstanding. Final delayed billing and post-recording cleanup remain open.
