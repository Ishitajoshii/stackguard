# Original cloud inline-tool proof

On 15 November 2026, the deployed AgentCore Harness in Mumbai completed a real
inline-tool handoff using the APAC Nova Micro inference profile. The
[original SDK stream](../captures/agentcore-public-inline-20261115/stream.jsonl)
contains a model-selected `github_repo` call for `golang/go`, a `tool_use` stop,
and an `end_turn` after the matching result was returned in the same session.
The local callback performed an actual anonymous GitHub API read. The final
[response](../captures/agentcore-public-inline-20261115/summary.json) reports
BSD-3-Clause, matching that read. Publication hashes are recorded alongside it.

This is public metadata only: no provider token, resume, or Notion content was
sent to the cloud. It proves service-role assumption, model access, managed
session memory, streaming tool selection and continuation. It does not prove a
Codex delegation, a CloudWatch export, or three-credential lineage.

The live checks exposed three configuration errors:

1. Execution trust needed the actual runtime ARN's endpoint/version suffixes.
2. The managed memory needed scoped ListEvents, GetEvent and CreateEvent access.
3. Inline tools require `@name` allowlist selectors. Plain names matched no inline
   tool: that run returned unverified prose without executing a callback and was
   rejected by the verifier. The regression test now checks each exact selector.

The SDK request was inspected offline with anonymous credentials and a stub HTTP
client before retrying. No wildcard or built-in shell access was enabled. The
bridge validates returned tool names against the configured definitions, rather
than against their differently formatted allowlist selectors.

AWS's [selector documentation](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/harness-tools.html)
and [Launchpad implementation notes](https://github.com/aws-samples/sample-agentcore-launchpad/blob/main/docs/architecture.zh-CN.md)
explain this distinction. The successful live stream is the local evidence for
the chosen selector. Earlier failed attempts remain private; they are included
in the cost ledger rather than counted as successful runs.
