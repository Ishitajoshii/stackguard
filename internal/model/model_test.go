package model

import (
	"encoding/json"
	"testing"
	"time"
)

// Schema/property inputs are not claimed as real agent captures.
func TestEventValidation(t *testing.T) {
	e := Event{Version: 1, ID: "event", TS: time.Now(), Source: "protocol-test", Session: "session", Span: "span", Agent: "test-client", Kind: "tool_call"}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	e.NHI = "raw-secret"
	if e.Validate() == nil {
		t.Fatal("raw value accepted as identity")
	}
	e.NHI = ""
	e.Version = 99
	if e.Validate() == nil {
		t.Fatal("unknown version accepted")
	}
}
func TestRecordWireNames(t *testing.T) {
	b, err := json.Marshal(Edge{Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if v["call_count"] != float64(2) {
		t.Fatal("wrong contract")
	}
}
