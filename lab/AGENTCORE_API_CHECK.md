# AgentCore API availability check

Checked 13 November 2026. This is an API-shape check, not a successful cloud invocation.

- Installed AWS CLI 2.38.5 recognizes `aws bedrock-agentcore-control create-harness --generate-cli-skeleton input` without making a network request. Its input includes model, tools, allowedTools, maxIterations, maxTokens and timeoutSeconds.
- The official Go control-plane package exposes CreateHarness and GetHarness: [package documentation](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol).
- The official Go data-plane source exposes InvokeHarness with a context and event-stream result: [AWS SDK source](https://github.com/aws/aws-sdk-go-v2/blob/main/service/bedrockagentcore/api_op_InvokeHarness.go). Observed source blob: `48df4db209e3f008c8e5e21b3f1d14eb8d9d0c5d`.
- The observed service go.mod requires Go 1.24: [module definition](https://github.com/aws/aws-sdk-go-v2/blob/main/service/bedrockagentcore/go.mod). The installed Go 1.26.5 satisfies it. Raise the consuming module's Go minimum when integrating a version with this requirement; the current standard-library-only CLI does not yet import it.

The [InvokeHarness API](https://docs.aws.amazon.com/bedrock-agentcore/latest/APIReference/API_InvokeHarness.html) requires a harness ARN, messages and runtime session ID. The bridge must consume stream events and preserve explicit correlation. Do not substitute a one-shot model call for the planned cloud agent.

Remaining proof: pin a released SDK version, compile the bridge against it, authenticate, select an available regional model, create and await a harness, complete an inline function continuation and retrieve native cloud evidence. Do not mark the cloud feasibility gate complete from these source checks.
