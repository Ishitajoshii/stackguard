package telemetry

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectorDropsContentAndAccounts(t *testing.T) {
	var saved any
	h := Handler(func(v any) error { saved = v; return nil })
	r := httptest.NewRequest("POST", "http://localhost/v1/logs", strings.NewReader(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"body":{"stringValue":"PRIVATE"},"attributes":[{"key":"event.name","value":{"stringValue":"codex.tool_result"}},{"key":"user.email","value":{"stringValue":"private@example.com"}},{"key":"arguments","value":{"stringValue":"PRIVATE"}}]}]}]}]}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	b, _ := json.Marshal(saved)
	if w.Code != 200 || strings.Contains(string(b), "PRIVATE") || strings.Contains(string(b), "private@example.com") || !strings.Contains(string(b), "codex.tool_result") {
		t.Fatal(w.Code, string(b))
	}
}

func TestCollectorRejectsUnknownEnvelopeContent(t *testing.T) {
	b, _ := json.Marshal(sanitize(map[string]any{"resourceLogs": []any{}, "futurePrompt": "PRIVATE", "scope": map[string]any{"name": "PRIVATE"}}, 0))
	if strings.Contains(string(b), "PRIVATE") {
		t.Fatal(string(b))
	}
}
