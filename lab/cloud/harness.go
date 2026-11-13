// Package cloud runs the independently configured AgentCore Harness. It does
// not replace the cloud agent with a scripted sequence of pretend tool calls.
package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"github.com/Ishitajoshii/stackguard/lab/mcp"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcore"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcore/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcore/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"sort"
	"strings"
	"time"
)

type Options struct {
	Profile, Region, Harness, Session string
	MaxRounds                         int
	ToolCall                          mcp.Call
	Audit                             func(any) error
	Events                            func(any) error
	ParentSession, ParentRequestID    string
}
type ToolRequest struct {
	ID, Name string
	Input    strings.Builder
}
type Round struct {
	Text  strings.Builder
	Tools map[int32]*ToolRequest
	Stop  string
	Usage []Usage
}

// Usage retains provider-reported metadata without guessing billing rates or
// treating missing metadata as zero usage. Each entry is one stream event.
type Usage struct {
	Input      int32  `json:"input_tokens"`
	Output     int32  `json:"output_tokens"`
	Total      int32  `json:"total_tokens"`
	CacheRead  *int32 `json:"cache_read_input_tokens,omitempty"`
	CacheWrite *int32 `json:"cache_write_input_tokens,omitempty"`
	LatencyMS  int64  `json:"latency_ms"`
}

func (r *Round) Consume(event types.InvokeHarnessStreamOutput) error {
	if r.Tools == nil {
		r.Tools = map[int32]*ToolRequest{}
	}
	switch v := event.(type) {
	case *types.InvokeHarnessStreamOutputMemberMetadata:
		u, m := v.Value.Usage, v.Value.Metrics
		if u == nil || m == nil || u.InputTokens == nil || u.OutputTokens == nil || u.TotalTokens == nil || m.LatencyMs == nil {
			return errors.New("cloud usage metadata lacks required fields")
		}
		if *u.InputTokens < 0 || *u.OutputTokens < 0 || *u.TotalTokens < 0 || aws.ToInt32(u.CacheReadInputTokens) < 0 || aws.ToInt32(u.CacheWriteInputTokens) < 0 || *m.LatencyMs < 0 || len(r.Usage) >= 64 {
			return errors.New("cloud usage metadata is invalid or exceeds limit")
		}
		r.Usage = append(r.Usage, Usage{Input: *u.InputTokens, Output: *u.OutputTokens, Total: *u.TotalTokens, CacheRead: u.CacheReadInputTokens, CacheWrite: u.CacheWriteInputTokens, LatencyMS: *m.LatencyMs})
	case *types.InvokeHarnessStreamOutputMemberContentBlockStart:
		if t, ok := v.Value.Start.(*types.HarnessContentBlockStartMemberToolUse); ok {
			if len(r.Tools) >= 8 {
				return errors.New("cloud tool batch exceeds eight")
			}
			index := aws.ToInt32(v.Value.ContentBlockIndex)
			if _, ok := r.Tools[index]; ok {
				return errors.New("duplicate cloud content block")
			}
			r.Tools[index] = &ToolRequest{ID: aws.ToString(t.Value.ToolUseId), Name: aws.ToString(t.Value.Name)}
		}
	case *types.InvokeHarnessStreamOutputMemberContentBlockDelta:
		switch d := v.Value.Delta.(type) {
		case *types.HarnessContentBlockDeltaMemberText:
			if r.Text.Len()+len(d.Value) > 128<<10 {
				return errors.New("cloud response exceeds limit")
			}
			r.Text.WriteString(d.Value)
		case *types.HarnessContentBlockDeltaMemberToolUse:
			t := r.Tools[aws.ToInt32(v.Value.ContentBlockIndex)]
			if t == nil {
				return errors.New("cloud delta has no tool start")
			}
			s := aws.ToString(d.Value.Input)
			if t.Input.Len()+len(s) > 64<<10 {
				return errors.New("cloud tool arguments exceed limit")
			}
			t.Input.WriteString(s)
		}
	case *types.InvokeHarnessStreamOutputMemberMessageStop:
		r.Stop = string(v.Value.StopReason)
	}
	return nil
}
func Definitions() ([]types.HarnessTool, []string) {
	var out []types.HarnessTool
	var allowed []string
	for _, t := range mcp.Tools() {
		out = append(out, types.HarnessTool{Type: types.HarnessToolTypeInlineFunction, Name: aws.String(t.Name), Config: &types.HarnessToolConfigurationMemberInlineFunction{Value: types.HarnessInlineFunctionConfig{Description: aws.String(t.Description), InputSchema: document.NewLazyDocument(t.Schema)}}})
		allowed = append(allowed, t.Name)
	}
	return out, allowed
}
func Run(ctx context.Context, o Options, prompt string) (string, error) {
	if o.Region == "" || !strings.HasPrefix(o.Harness, "arn:aws:bedrock-agentcore:") || len(o.Session) < 33 || len(o.Session) > 100 || o.ToolCall == nil {
		return "", errors.New("configure region, harness ARN, 33..100 character session and local tool endpoint")
	}
	if len(prompt) > 64<<10 {
		return "", errors.New("normalized prompt exceeds 64 KiB")
	}
	if o.MaxRounds < 1 || o.MaxRounds > 8 {
		return "", errors.New("cloud rounds must be 1..8")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cfg, e := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(o.Profile), config.WithRegion(o.Region), config.WithRetryMaxAttempts(1))
	if e != nil {
		return "", errors.New("AWS configuration unavailable; configure a restricted profile")
	}
	client := bedrockagentcore.NewFromConfig(cfg)
	identity, identityErr := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if identityErr != nil {
		return "", errors.New("AWS caller identity unavailable; refresh the restricted lab profile")
	}
	if !strings.Contains(aws.ToString(identity.Arn), ":assumed-role/") {
		return "", errors.New("cloud research requires an assumed lab role; root and long-term IAM user credentials are rejected")
	}
	if o.Events != nil {
		if e = o.Events(model.Event{Version: 1, ID: o.Session + ":root", TS: time.Now().UTC(), Source: "bridge", Session: o.Session, Span: o.Session, ParentSession: o.ParentSession, ParentRequestID: o.ParentRequestID, Agent: "agentcore-researcher", Client: "local-bridge", Kind: "delegation", Evidence: []string{"bridge:" + o.Session}}); e != nil {
			return "", e
		}
	}
	definitions, allowed := Definitions()
	messages := []types.HarnessMessage{{Role: types.HarnessConversationRole("user"), Content: []types.HarnessContentBlock{&types.HarnessContentBlockMemberText{Value: prompt}}}}
	seen := map[string]bool{}
	var final strings.Builder
	for n := 0; n < o.MaxRounds; n++ {
		if o.Events != nil {
			credentials, e := cfg.Credentials.Retrieve(ctx)
			if e != nil {
				return final.String(), errors.New("AWS role credentials unavailable")
			}
			credentialID, _ := secrets.AWSSet(credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken)
			span := fmt.Sprintf("%s:invoke:%d", o.Session, n)
			if e = o.Events(model.Event{Version: 1, ID: span, TS: time.Now().UTC(), Source: "bridge", Session: o.Session, Span: span, ParentSession: o.ParentSession, ParentRequestID: o.ParentRequestID, Agent: "local-bridge", Client: "aws-sdk", Kind: "tool_call", Target: "AgentCore.InvokeHarness", Status: "attempted", Credentials: []model.CredentialUse{{ID: credentialID, Attribution: "implicit"}, {ID: "aws_role:" + aws.ToString(identity.Arn), Attribution: "implicit"}}, Evidence: []string{"aws-sdk-credential-provider", "sts:GetCallerIdentity"}}); e != nil {
				return final.String(), e
			}
		}
		response, e := client.InvokeHarness(ctx, &bedrockagentcore.InvokeHarnessInput{HarnessArn: aws.String(o.Harness), RuntimeSessionId: aws.String(o.Session), Messages: messages, Tools: definitions, AllowedTools: allowed, MaxIterations: aws.Int32(6), MaxTokens: aws.Int32(2048), TimeoutSeconds: aws.Int32(60)})
		if e != nil {
			if ctx.Err() != nil {
				return final.String(), ctx.Err()
			}
			return final.String(), errors.New("AgentCore invocation failed; verify READY state, region, model access and restricted role permissions")
		}
		stream := response.GetStream()
		if stream == nil {
			return final.String(), errors.New("AgentCore returned no stream")
		}
		round := Round{}
		for event := range stream.Events() {
			if e = round.Consume(event); e != nil {
				stream.Close()
				return final.String(), e
			}
		}
		e = stream.Err()
		stream.Close()
		if e != nil {
			return final.String(), errors.New("AgentCore stream failed; retain partial audit and retry in a new session")
		}
		if final.Len()+round.Text.Len() > 128<<10 {
			return final.String(), errors.New("cloud narrative exceeds limit")
		}
		final.WriteString(round.Text.String())
		if o.Audit != nil {
			if e = o.Audit(map[string]any{"source": "agentcore-sdk", "session": o.Session, "round": n, "stop_reason": round.Stop, "tool_requests": len(round.Tools), "usage_available": len(round.Usage) > 0, "usage_events": round.Usage, "checked_at": time.Now().UTC()}); e != nil {
				return final.String(), e
			}
		}
		if round.Stop == "end_turn" {
			return secrets.Redact(final.String()), nil
		}
		if round.Stop != "tool_use" || len(round.Tools) == 0 {
			return final.String(), errors.New("cloud stopped before a complete result or valid tool handoff")
		}
		indices := make([]int, 0, len(round.Tools))
		for k := range round.Tools {
			indices = append(indices, int(k))
		}
		sort.Ints(indices)
		var content []types.HarnessContentBlock
		for _, index := range indices {
			tool := round.Tools[int32(index)]
			if tool.ID == "" || seen[tool.ID] {
				return final.String(), errors.New("missing or replayed cloud tool-use ID")
			}
			seen[tool.ID] = true
			known := false
			for _, name := range allowed {
				if name == tool.Name {
					known = true
				}
			}
			if !known {
				return final.String(), errors.New("cloud requested a tool outside the allowlist")
			}
			args := tool.Input.String()
			if args == "" {
				args = "{}"
			}
			if !json.Valid([]byte(args)) {
				return final.String(), errors.New("invalid cloud tool arguments")
			}
			if o.Events != nil {
				if e = o.Events(model.Event{Version: 1, ID: tool.ID, TS: time.Now().UTC(), Source: "bridge", Session: o.Session, Span: tool.ID, Parent: o.Session, Agent: "agentcore-researcher", Client: "local-bridge", Kind: "tool_call", Target: tool.Name, Status: "attempted", Evidence: []string{"agentcore-sdk-tool-handoff:" + tool.ID}}); e != nil {
					return final.String(), e
				}
			}
			result, callErr := o.ToolCall(mcp.WithCorrelation(ctx, mcp.Correlation{Session: o.Session, Span: tool.ID}), tool.Name, json.RawMessage(args))
			status := types.HarnessToolUseStatusSuccess
			if callErr != nil {
				status = types.HarnessToolUseStatus("error")
				result = map[string]string{"error": "read-only tool failed; check credentials, query budget and supplied identifiers"}
			}
			b, e := json.Marshal(result)
			if e != nil || len(b) > 512<<10 {
				return final.String(), errors.New("cloud tool result exceeds limit")
			}
			if o.Audit != nil {
				if e = o.Audit(map[string]any{"source": "bridge", "session": o.Session, "cloud_tool_use_id": tool.ID, "tool": tool.Name, "status": string(status), "result_sha256": secrets.Fingerprint(string(b)), "checked_at": time.Now().UTC()}); e != nil {
					return final.String(), e
				}
			}
			content = append(content, &types.HarnessContentBlockMemberToolResult{Value: types.HarnessToolResultBlock{ToolUseId: aws.String(tool.ID), Status: status, Content: []types.HarnessToolResultContentBlock{&types.HarnessToolResultContentBlockMemberText{Value: secrets.Redact(string(b))}}}})
		}
		messages = []types.HarnessMessage{{Role: types.HarnessConversationRole("user"), Content: content}}
	}
	return secrets.Redact(final.String()), fmt.Errorf("cloud handoff limit reached after %d rounds", o.MaxRounds)
}
