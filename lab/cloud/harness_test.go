package cloud

import (
	"encoding/hex"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcore/types"
	"testing"
)

func TestNativeTraceContext(t *testing.T) {
	trace, parent, header, err := traceContext()
	if err != nil || len(trace) != 32 || len(parent) != 16 || header != "00-"+trace+"-"+parent+"-01" {
		t.Fatal("invalid W3C trace context", err)
	}
	if _, err := hex.DecodeString(trace + parent); err != nil {
		t.Fatal(err)
	}
	other, _, _, err := traceContext()
	if err != nil || trace == other {
		t.Fatal("invocations must not reuse trace roots")
	}
}

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
	for i, s := range allowed {
		if s != "@"+aws.ToString(tools[i].Name) {
			t.Fatal("inline tool selector must be @name")
		}
	}
}

func TestUsageMetadataRetainsReportedCountsAndAbsence(t *testing.T) {
	r := Round{}
	if len(r.Usage) != 0 {
		t.Fatal("missing usage became zero-valued evidence")
	}
	event := &types.InvokeHarnessStreamOutputMemberMetadata{Value: types.HarnessMetadataEvent{
		Usage:   &types.HarnessTokenUsage{InputTokens: aws.Int32(120), OutputTokens: aws.Int32(20), TotalTokens: aws.Int32(140), CacheReadInputTokens: aws.Int32(100)},
		Metrics: &types.HarnessStreamMetrics{LatencyMs: aws.Int64(250)},
	}}
	if err := r.Consume(event); err != nil {
		t.Fatal(err)
	}
	if len(r.Usage) != 1 || r.Usage[0].Total != 140 || r.Usage[0].CacheRead == nil || *r.Usage[0].CacheRead != 100 || r.Usage[0].CacheWrite != nil || r.Usage[0].LatencyMS != 250 {
		t.Fatal("reported usage changed")
	}
	event.Value.Usage.InputTokens = aws.Int32(-1)
	if err := r.Consume(event); err == nil {
		t.Fatal("negative usage accepted")
	}
	if err := r.Consume(&types.InvokeHarnessStreamOutputMemberMetadata{}); err == nil {
		t.Fatal("missing required usage fields accepted")
	}
}
