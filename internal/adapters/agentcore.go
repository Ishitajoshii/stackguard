package adapters

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var spanIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// AgentCoreSpan reads the flat span JSON actually exported by CloudWatch
// Transaction Search. Unknown attributes (including auth IDs and payloads) are
// deliberately excluded. An OTLP envelope is handled by the separate adapter.
type AgentCoreSpan struct{}

func (AgentCoreSpan) Decode(b []byte) ([]model.Event, error) {
	var v struct {
		Trace      string                     `json:"traceId"`
		Span       string                     `json:"spanId"`
		Parent     string                     `json:"parentSpanId"`
		Start      json.Number                `json:"startTimeUnixNano"`
		Attributes map[string]json.RawMessage `json:"attributes"`
		Status     struct {
			Code string `json:"code"`
		} `json:"status"`
	}
	if json.Unmarshal(b, &v) != nil || !traceIDPattern.MatchString(v.Trace) || !spanIDPattern.MatchString(v.Span) || v.Parent != "" && !spanIDPattern.MatchString(v.Parent) {
		return nil, errors.New("invalid native AgentCore span identifiers")
	}
	nano, err := strconv.ParseInt(v.Start.String(), 10, 64)
	if err != nil || nano <= 0 {
		return nil, errors.New("invalid native AgentCore span timestamp")
	}
	get := func(key string) string { var s string; _ = json.Unmarshal(v.Attributes[key], &s); return s }
	agent := get("gen_ai.agent.name")
	if agent == "" {
		agent = "unknown-cloud-agent"
	}
	kind, target := "span", get("gen_ai.tool.name")
	if target != "" {
		kind = "tool_call"
	}
	status := "unknown"
	switch get("gen_ai.tool.status") {
	case "success":
		status = "succeeded"
	case "error":
		status = "failed"
	}
	if v.Status.Code == "ERROR" {
		status = "failed"
	} else if status == "unknown" && v.Status.Code == "OK" {
		status = "succeeded"
	}
	if v.Parent == "0000000000000000" {
		v.Parent = ""
	}
	e := model.Event{Version: 1, ID: secrets.Fingerprint(v.Trace + ":" + v.Span), TS: time.Unix(0, nano).UTC(), Source: "agentcore-native", Session: v.Trace, Span: v.Span, Parent: v.Parent, Agent: agent, Kind: kind, Target: secrets.Redact(target), CallID: get("gen_ai.tool.call.id"), Status: status, Attribution: "unknown"}
	if session := get("session.id"); session != "" {
		e.Evidence = []string{"agentcore-session:" + session}
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return []model.Event{e}, nil
}
