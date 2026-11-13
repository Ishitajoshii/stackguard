package adapters

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeCodexMetadataOnly(t *testing.T) {
	b := []byte(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"attributes":[{"key":"event.name","value":{"stringValue":"codex.tool_result"}},{"key":"event.timestamp","value":{"stringValue":"2026-01-01T00:00:00Z"}},{"key":"conversation.id","value":{"stringValue":"session"}},{"key":"call_id","value":{"stringValue":"call"}},{"key":"tool_name","value":{"stringValue":"read"}},{"key":"success","value":{"boolValue":true}},{"key":"arguments","value":{"stringValue":"PRIVATE_CONTENT"}}]}]}]}]}`)
	events, e := (Codex{}).Decode(b)
	if e != nil || len(events) != 1 || events[0].Status != "succeeded" {
		t.Fatal(events, e)
	}
	encoded, _ := json.Marshal(events)
	if strings.Contains(string(encoded), "PRIVATE_CONTENT") {
		t.Fatal("copied arguments")
	}
}
