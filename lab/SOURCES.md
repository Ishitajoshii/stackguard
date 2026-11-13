# Research source index

Reviewed 2026-11-13. Links support documented behavior; none establishes that a live lab invocation occurred. Algorithm fixtures are not original captures.

| Primary source | Claim supported | Matching implementation/evidence |
|---|---|---|
| [MCP transports, revision 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports) | stdio framing, Streamable HTTP, session and Origin handling | internal/proxy; real file transport captures in results/bench |
| [Codex configuration](https://learn.chatgpt.com/docs/config-file/config-reference) | MCP environment/header references and OTLP JSON export | internal/config; lab/examples/codex.toml; native capture pending |
| [Codex security](https://learn.chatgpt.com/docs/agent-approvals-security) | Opt-in telemetry and prompt-content controls | Metadata-only collector; native validation pending |
| [Codex tool event source](https://github.com/openai/codex/blob/main/codex-rs/otel/src/tool_result.rs) | Tool-result identifiers and event fields | Source blob 7f9c3ca1419d1bb3f56a7c6854c152dc81f6aa8b inspected; codex-otlp adapter |
| [AgentCore inline tools](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-tools.html) | Streamed handoff and matching tool-result continuation | lab/cloud; contract tests, no live invocation |
| [InvokeHarness API](https://docs.aws.amazon.com/bedrock-agentcore/latest/APIReference/API_InvokeHarness.html) | Runtime session and invocation limits | SDK v1.58.0, compiled bridge |
| [CreateHarness API](https://docs.aws.amazon.com/bedrock-agentcore-control/latest/APIReference/API_CreateHarness.html) | Reviewable deployment inputs | cloud-plan and deploy-harness.ps1 |
| [Harness security](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-security.html) | Separate execution role, caller permissions and tool-selection boundaries | Assumed-role guard and explicit tool allowlist; no cloud proof |
| [Temporary AWS credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_request.html) | Access key, secret and session token form one temporary credential | AWSSet algorithm tests |
| [STS GetCallerIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html) | Caller account/ARN identity, not model authorization | Live use deferred; bridge guard |
| [GitHub secret-scanning patterns](https://docs.github.com/en/code-security/reference/secret-security/supported-secret-scanning-patterns) | Provider/type distinction and validity-check limitations | Candidate heuristics; not a claim to reproduce GitHub's private detectors |
| [GitHub issue timelines](https://docs.github.com/en/rest/issues/timeline?apiVersion=2022-11-28) | Cross-referenced issue/PR events | Bounded linked-PR checks |
| [GitHub community profiles](https://docs.github.com/en/rest/metrics/community?apiVersion=2022-11-28) | Contribution-guide metadata | Onboarding score evidence |
| [Notion authentication](https://developers.notion.com/reference/authentication) | Bearer authentication and integration permissions | Dedicated token environment reference |
| [Notion block children](https://developers.notion.com/reference/get-block-children) | Paginated block reads | One configured subtree, bounded depth/count |
| [OpenAI authentication](https://platform.openai.com/docs/api-reference/authentication) | Bearer API key; organization/project context | Candidate detector; no paid verifier |
| [Anthropic API overview](https://platform.claude.com/docs/en/api/overview) | API authentication/version headers | Candidate detector; manual read-only verification route |
| [Slack auth.test](https://docs.slack.dev/reference/methods/auth.test/) | Read-only token identity check | Documented option; no automatic Slack verifier |
| [Google API-key practices](https://docs.cloud.google.com/docs/authentication/api-keys-best-practices) | Service restrictions, key handling | Context/format caution; no universal verification |
| [Azure OpenAI authentication](https://learn.microsoft.com/azure/api-management/api-management-authenticate-authorize-azure-openai) | Resource-specific api-key or managed identity | Contextual Azure detector |
| [Claude Code MCP](https://code.claude.com/docs/en/mcp) | MCP scopes, env expansion, remote authentication | Research only, not observed in this lab |
| [Cursor MCP](https://cursor.com/docs/mcp) | Project/user config, variable interpolation and remote auth | Research only, not observed in this lab |
| [Copilot CLI MCP](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers) | CLI configuration and MCP servers | Research only, not observed in this lab |
