# AgentCore API availability check

Checked 13 November 2026. This is an API-shape check, not a successful cloud invocation.

- Installed AWS CLI 2.38.5 recognizes `aws bedrock-agentcore-control create-harness --generate-cli-skeleton input` without making a network request. Its input includes model, tools, allowedTools, maxIterations, maxTokens and timeoutSeconds.
- The official Go control-plane package exposes CreateHarness and GetHarness: [package documentation](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol).
- The official Go data-plane source exposes InvokeHarness with a context and event-stream result: [AWS SDK source](https://github.com/aws/aws-sdk-go-v2/blob/main/service/bedrockagentcore/api_op_InvokeHarness.go). Observed source blob: `48df4db209e3f008c8e5e21b3f1d14eb8d9d0c5d`.
- The observed service go.mod requires Go 1.24: [module definition](https://github.com/aws/aws-sdk-go-v2/blob/main/service/bedrockagentcore/go.mod). The installed Go 1.26.5 satisfies it. The consuming module now declares Go 1.24 and compiles the bridge with the official data-plane SDK v1.58.0.

The [InvokeHarness API](https://docs.aws.amazon.com/bedrock-agentcore/latest/APIReference/API_InvokeHarness.html) requires a harness ARN, messages and runtime session ID. The bridge must consume stream events and preserve explicit correlation. Do not substitute a one-shot model call for the planned cloud agent.

Completed at the original checkpoint: released SDK pin, streaming union-type compilation and continuation contract tests. On 15 November, the [live public handoff](CLOUD_HANDOFF_PROOF.md) verified authentication, model access, READY deployment and inline continuation. The later [native research proof](NATIVE_SCOUT_PROOF.md) verifies cloud telemetry and credential-bearing delegation, with source gaps explicitly disclosed.
