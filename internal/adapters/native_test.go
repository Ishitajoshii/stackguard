package adapters

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOriginalCodexFixtureProvenance(t *testing.T) {
	var provenance struct {
		Source string `json:"source"`
		SHA256 string `json:"source_sha256"`
		Line   int    `json:"source_jsonl_line"`
	}
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if err := json.Unmarshal(read("testdata/codex-file-tool.provenance.json"), &provenance); err != nil {
		t.Fatal(err)
	}
	source := read(filepath.Join("../..", provenance.Source))
	if fmt.Sprintf("%x", sha256.Sum256(source)) != provenance.SHA256 {
		t.Fatal("original capture checksum changed")
	}
	lines := strings.Split(string(source), "\n")
	if provenance.Line < 1 || provenance.Line > len(lines) {
		t.Fatal("invalid source line")
	}
	records := func(b []byte) []any {
		t.Helper()
		var envelope struct {
			Resources []struct {
				Scopes []struct {
					Records []any `json:"logRecords"`
				} `json:"scopeLogs"`
			} `json:"resourceLogs"`
		}
		if err := json.Unmarshal(b, &envelope); err != nil {
			t.Fatal(err)
		}
		var result []any
		for _, resource := range envelope.Resources {
			for _, scope := range resource.Scopes {
				result = append(result, scope.Records...)
			}
		}
		return result
	}
	fixture := records(read("testdata/codex-file-tool.otlp.json"))
	if len(fixture) != 1 {
		t.Fatal("expected exactly one selected original record")
	}
	for _, original := range records([]byte(lines[provenance.Line-1])) {
		if reflect.DeepEqual(fixture[0], original) {
			return
		}
	}
	t.Fatal("fixture record is not an unchanged record from the stated source line")
}

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

func TestOriginalCodexToolCapture(t *testing.T) {
	b, err := os.ReadFile("testdata/codex-file-tool.otlp.json")
	if err != nil {
		t.Fatal(err)
	}
	events, err := (Codex{}).Decode(b)
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	e := events[0]
	if e.Session != "01a1d639-5451-7fd1-ab82-f45ac735a6ce" || e.CallID != "exec-f3ba7320-30c1-45c2-b8e2-efa16017d208" || e.Target != "repository_file" || e.Status != "succeeded" || e.TS.Format("2006-01-02T15:04:05.000Z") != "2026-11-13T18:44:09.710Z" {
		t.Fatal("original native identifiers or outcome were changed", e)
	}
	if e.NHI != "" || e.Parent != "" || e.Attribution != "unknown" {
		t.Fatal("inferred unsupported credential or delegation", e)
	}
}
