package adapters

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOriginalAgentCoreFlatSpan(t *testing.T) {
	b, err := os.ReadFile("testdata/agentcore-shortlist.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := (AgentCoreSpan{}).Decode(b)
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	e := rows[0]
	if e.Session != "869646dbc136e2864c22033fa15ec4a2" || e.Span != "b7ff09eb53aa6752" || e.Parent != "c4c1ca278071063c" || e.TS.UnixNano() != 1794758230806233042 || e.CallID != "tooluse_29tp3VLflimBpI7X9W82Sp" || e.Target != "github_shortlist" || e.Status != "succeeded" {
		t.Fatal("original native fields changed", e)
	}
	if e.NHI != "" || e.Attribution != "unknown" || e.Agent != "unknown-cloud-agent" {
		t.Fatal("inferred unsupported identity", e)
	}
	var raw map[string]any
	json.Unmarshal(b, &raw)
	raw["attributes"].(map[string]any)["aws.auth.account.access_key"] = "PRIVATE_AUTH_ID"
	raw["attributes"].(map[string]any)["gen_ai.tool.input"] = "PRIVATE_CONTENT"
	modified, _ := json.Marshal(raw)
	decoded, err := (AgentCoreSpan{}).Decode(modified)
	encoded, _ := json.Marshal(decoded)
	if err != nil || strings.Contains(string(encoded), "PRIVATE_") {
		t.Fatal("private native attributes copied", err)
	}
	raw["traceId"] = "not-a-trace"
	modified, _ = json.Marshal(raw)
	if _, err = (AgentCoreSpan{}).Decode(modified); err == nil {
		t.Fatal("invalid native ID accepted")
	}
}

func TestOriginalAgentCoreFixtureProvenance(t *testing.T) {
	var p struct {
		Source string `json:"source"`
		SHA    string `json:"source_sha256"`
		Index  int    `json:"event_index"`
		ID     string `json:"event_id"`
	}
	b, err := os.ReadFile("testdata/agentcore-shortlist.provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("../..", p.Source))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(source)) != p.SHA {
		t.Fatal("source checksum changed")
	}
	var export struct {
		Events []struct {
			ID      string `json:"eventId"`
			Message string `json:"message"`
		} `json:"events"`
	}
	if err = json.Unmarshal(source, &export); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("testdata/agentcore-shortlist.json")
	if err != nil {
		t.Fatal(err)
	}
	if p.Index >= len(export.Events) || export.Events[p.Index].ID != p.ID || export.Events[p.Index].Message != strings.TrimSpace(string(fixture)) {
		t.Fatal("fixture is not the original published span")
	}
	rows, err := (CloudWatch{}).Decode(source)
	if err != nil || len(rows) != 44 {
		t.Fatal("native CloudWatch envelope failed", len(rows), err)
	}
}
