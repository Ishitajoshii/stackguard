// Package telemetry collects metadata-only OTLP JSON. It rejects protobuf;
// configure the sender's OTLP/HTTP JSON exporter explicitly.
package telemetry

import (
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"io"
	"net/http"
	"strings"
)

var allowed = map[string]bool{"event.name": true, "event.timestamp": true, "conversation.id": true, "call_id": true, "tool_result_seq": true, "tool_name": true, "tool_namespace": true, "mcp_server": true, "mcp_server_origin": true, "duration_ms": true, "success": true, "originator": true, "app.version": true, "service.name": true, "service.version": true, "gen_ai.agent.name": true, "gen_ai.tool.name": true, "gen_ai.tool.call.id": true, "gen_ai.operation.name": true, "aws.bedrock.agentcore.session_id": true}

func sanitize(v any, depth int) any {
	if depth > 32 {
		return nil
	}
	switch x := v.(type) {
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, sanitize(item, depth+1))
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for key, value := range x {
			switch key {
			case "body", "status":
				continue
			case "attributes":
				items, _ := value.([]any)
				var attrs []any
				for _, item := range items {
					a, ok := item.(map[string]any)
					if !ok {
						continue
					}
					name, _ := a["key"].(string)
					if allowed[name] {
						attrs = append(attrs, sanitize(a, depth+1))
					}
				}
				out[key] = attrs
			default:
				// Unknown envelope fields can contain prompts or account data too.
				switch key {
				case "resourceLogs", "resourceSpans", "resource", "scopeLogs", "scopeSpans", "scope", "logRecords", "spans", "timeUnixNano", "observedTimeUnixNano", "startTimeUnixNano", "endTimeUnixNano", "traceId", "spanId", "parentSpanId", "flags", "key", "value", "stringValue", "boolValue", "intValue", "doubleValue":
					out[key] = sanitize(value, depth+1)
				}
			}
		}
		return out
	case string:
		return secrets.Redact(x)
	default:
		return v
	}
}
func Handler(write func(any) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/logs" && r.URL.Path != "/v1/traces" {
			http.Error(w, "OTLP JSON POST endpoint required", 404)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "configure OTLP/HTTP protocol=json; protobuf is not accepted", 415)
			return
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if e != nil || len(b) > 1<<20 {
			http.Error(w, "OTLP envelope exceeds 1 MiB", 413)
			return
		}
		var v map[string]any
		if json.Unmarshal(b, &v) != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if _, logs := v["resourceLogs"]; !logs {
			if _, spans := v["resourceSpans"]; !spans {
				http.Error(w, "OTLP logs or spans required", 400)
				return
			}
		}
		if e = write(sanitize(v, 0)); e != nil {
			http.Error(w, "capture persistence failed", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "{}")
	})
}
