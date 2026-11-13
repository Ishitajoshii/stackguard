package adapters

import (
	"encoding/json"
	"errors"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"strconv"
	"time"
)

type attribute struct {
	Key   string `json:"key"`
	Value struct {
		String string `json:"stringValue"`
		Bool   *bool  `json:"boolValue"`
		Int    string `json:"intValue"`
	} `json:"value"`
}

func attrs(list []attribute) map[string]string {
	out := map[string]string{}
	for _, a := range list {
		v := a.Value.String
		if a.Value.Bool != nil {
			v = strconv.FormatBool(*a.Value.Bool)
		}
		if a.Value.Int != "" {
			v = a.Value.Int
		}
		out[a.Key] = v
	}
	return out
}

// Codex decodes the native OTLP JSON log envelope using fields inspected in
// openai/codex's tool_result.rs and events/shared.rs. Prompt, arguments, outputs,
// account IDs and email fields are deliberately not copied to normalized data.
type Codex struct{}

func (Codex) Decode(b []byte) ([]model.Event, error) {
	var v struct {
		ResourceLogs []struct {
			ScopeLogs []struct {
				Records []struct {
					Time  string      `json:"timeUnixNano"`
					Attrs []attribute `json:"attributes"`
				} `json:"logRecords"`
			} `json:"scopeLogs"`
		} `json:"resourceLogs"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil, errors.New("invalid native Codex OTLP JSON")
	}
	var out []model.Event
	for _, r := range v.ResourceLogs {
		for _, scope := range r.ScopeLogs {
			for _, record := range scope.Records {
				a := attrs(record.Attrs)
				if a["event.name"] != "codex.tool_result" {
					continue
				}
				ts, e := time.Parse(time.RFC3339Nano, a["event.timestamp"])
				if e != nil {
					nano, err := strconv.ParseInt(record.Time, 10, 64)
					if err != nil {
						return nil, errors.New("Codex tool result has no valid timestamp")
					}
					ts = time.Unix(0, nano).UTC()
				}
				session, call := a["conversation.id"], a["call_id"]
				if session == "" || call == "" {
					return nil, errors.New("Codex tool result lacks explicit conversation/call ID")
				}
				status := "unknown"
				if a["success"] == "true" {
					status = "succeeded"
				} else if a["success"] == "false" {
					status = "failed"
				}
				evidence := model.Event{Version: 1, ID: secrets.Fingerprint(session + ":" + call + ":" + a["tool_result_seq"]), TS: ts, Source: "codex-native", Session: session, Span: call, CallID: call, Agent: "codex", Kind: "tool_call", Status: status, Target: secrets.Redact(a["tool_name"]), Attribution: "unknown", Client: a["originator"]}
				if e = evidence.Validate(); e != nil {
					return nil, e
				}
				out = append(out, evidence)
			}
		}
	}
	return out, nil
}

// OTLPSpans decodes cloud trace exports independently of provider-specific log
// bodies. traceId is the trace session; application runtime-session correlation
// still requires explicit bridge/native evidence, never timestamp proximity.
type OTLPSpans struct{}

func (OTLPSpans) Decode(b []byte) ([]model.Event, error) {
	var v struct {
		ResourceSpans []struct {
			Scopes []struct {
				Spans []struct {
					Trace  string      `json:"traceId"`
					Span   string      `json:"spanId"`
					Parent string      `json:"parentSpanId"`
					Name   string      `json:"name"`
					Start  string      `json:"startTimeUnixNano"`
					Attrs  []attribute `json:"attributes"`
				} `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil, errors.New("invalid OTLP trace JSON")
	}
	var out []model.Event
	for _, r := range v.ResourceSpans {
		for _, scope := range r.Scopes {
			for _, span := range scope.Spans {
				a := attrs(span.Attrs)
				agent := a["gen_ai.agent.name"]
				if agent == "" {
					agent = "unknown-cloud-agent"
				}
				nano, e := strconv.ParseInt(span.Start, 10, 64)
				if e != nil {
					return nil, errors.New("invalid cloud span timestamp")
				}
				kind := "span"
				target := a["gen_ai.tool.name"]
				if target != "" {
					kind = "tool_call"
				}
				record := model.Event{Version: 1, ID: secrets.Fingerprint(span.Trace + ":" + span.Span), TS: time.Unix(0, nano).UTC(), Source: "cloud-otlp", Session: span.Trace, Span: span.Span, Parent: span.Parent, Agent: agent, Kind: kind, Target: secrets.Redact(target), CallID: a["gen_ai.tool.call.id"], Attribution: "unknown"}
				if e = record.Validate(); e != nil {
					return nil, e
				}
				out = append(out, record)
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no OTLP spans; native log messages need their own schema adapter")
	}
	return out, nil
}
