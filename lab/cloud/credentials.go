package cloud

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/processcreds"
)

// MCP owns stdin. Never let the AWS SDK's default shell-based credential helper
// inherit it: on Windows a surviving helper can also keep a shell's pipes open.
// AWS CLI resolves the selected profile, including browser login and role chains.
func credentialCommand(ctx context.Context, program string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin = nil         // null device, not the JSON-RPC stream or an interactive prompt
	cmd.Stderr = io.Discard // arbitrary helper diagnostics can contain credentials
	cmd.WaitDelay = 500 * time.Millisecond
	return cmd
}

func profileCredentials(profile string) aws.CredentialsProvider {
	provider := processcreds.NewProviderCommand(processcreds.NewCommandBuilderFunc(func(ctx context.Context) (*exec.Cmd, error) {
		return credentialCommand(ctx, "aws", "configure", "export-credentials", "--profile", profile, "--format", "process", "--no-cli-pager"), nil
	}), func(o *processcreds.Options) { o.Timeout = 30 * time.Second })
	return aws.NewCredentialsCache(provider)
}
