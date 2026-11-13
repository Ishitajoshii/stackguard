package cloud

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcore/types"
	"testing"
)

// Wire-structure tests use algorithm inputs, not fabricated cloud-run evidence.
func TestFragmentedInlineTool(t *testing.T) {
	r := Round{}
	events := []types.InvokeHarnessStreamOutput{&types.InvokeHarnessStreamOutputMemberContentBlockStart{Value: types.HarnessContentBlockStartEvent{ContentBlockIndex: aws.Int32(0), Start: &types.HarnessContentBlockStartMemberToolUse{Value: types.HarnessToolUseBlockStart{Name: aws.String("github_repo"), ToolUseId: aws.String("tool-1")}}}}, &types.InvokeHarnessStreamOutputMemberContentBlockDelta{Value: types.HarnessContentBlockDeltaEvent{ContentBlockIndex: aws.Int32(0), Delta: &types.HarnessContentBlockDeltaMemberToolUse{Value: types.HarnessToolUseBlockDelta{Input: aws.String(`{"repository":`)}}}}, &types.InvokeHarnessStreamOutputMemberContentBlockDelta{Value: types.HarnessContentBlockDeltaEvent{ContentBlockIndex: aws.Int32(0), Delta: &types.HarnessContentBlockDeltaMemberToolUse{Value: types.HarnessToolUseBlockDelta{Input: aws.String(`"org/repo"}`)}}}}, &types.InvokeHarnessStreamOutputMemberMessageStop{Value: types.HarnessMessageStopEvent{StopReason: types.HarnessStopReasonToolUse}}}
	for _, e := range events {
		if err := r.Consume(e); err != nil {
			t.Fatal(err)
		}
	}
	if r.Tools[0].Input.String() != `{"repository":"org/repo"}` || r.Stop != "tool_use" {
		t.Fatal(r)
	}
	tools, allowed := Definitions()
	if len(tools) != 7 || len(allowed) != 7 {
		t.Fatal("unbounded tool set")
	}
	for _, s := range allowed {
		if s == "*" || s == "shell" {
			t.Fatal("unsafe default")
		}
	}
}
