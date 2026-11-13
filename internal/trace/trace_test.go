package trace

import (
	"context"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLineageDedupAndCounts(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "algorithm.jsonl")
	e := model.Event{Version: 1, ID: "child", TS: time.Unix(1, 0), Source: "algorithm-test", Session: "s", Span: "child", Parent: "parent", Agent: "tool", Kind: "tool_call", NHI: secrets.Fingerprint("test"), Attribution: "direct"}
	parent := e
	parent.ID = "parent"
	parent.Span = "parent"
	parent.Parent = ""
	parent.Agent = "researcher"
	parent.Kind = "delegation"
	parent.NHI = ""
	complete := e
	complete.ID = "done"
	complete.Kind = "tool_completion"
	f, _ := os.Create(p)
	enc := json.NewEncoder(f)
	enc.Encode(e)
	enc.Encode(e)
	enc.Encode(complete)
	enc.Encode(parent)
	f.Close()
	var previous, previousTree string
	for _, workers := range []int{1, 4, 8, 16} {
		out := filepath.Join(d, strings.Repeat("w", workers))
		m, err := Run(context.Background(), Options{Inputs: []Input{{"normalized", p}}, Output: out, Temp: d, Workers: workers, Lineage: true})
		if err != nil || !m.Complete || m.Records != 3 {
			t.Fatal(m, err)
		}
		b, _ := os.ReadFile(filepath.Join(out, "edges.jsonl"))
		var edge model.Edge
		if err = json.Unmarshal(b, &edge); err != nil || edge.Count != 1 || strings.Join(edge.Path, "/") != "researcher/tool" {
			t.Fatal(string(b), err)
		}
		if previous != "" && previous != string(b) {
			t.Fatal("worker drift")
		}
		previous = string(b)
		tree, err := os.ReadFile(filepath.Join(out, "lineage.txt"))
		if err != nil || !strings.Contains(string(tree), "`-- \"researcher\"\n    `-- \"tool\"") || !strings.Contains(string(tree), "attribution=direct") || !strings.Contains(string(tree), "evidence=") {
			t.Fatal("missing lineage tree or evidence", err)
		}
		if previousTree != "" && previousTree != string(tree) {
			t.Fatal("lineage tree changes with worker count")
		}
		previousTree = string(tree)
	}
}

func TestRiskFailurePreventsCompleteManifest(t *testing.T) {
	d := t.TempDir()
	input := filepath.Join(d, "empty.jsonl")
	out := filepath.Join(d, "result")
	os.WriteFile(input, nil, 0600)
	m, e := Run(context.Background(), Options{Inputs: []Input{{"normalized", input}}, Output: out, Temp: d, Workers: 1, Inventory: filepath.Join(d, "missing.jsonl")})
	if e == nil || m.Complete {
		t.Fatal("risk failure reported complete", m, e)
	}
	b, e := os.ReadFile(filepath.Join(out, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	if json.Unmarshal(b, &m) != nil || m.Complete {
		t.Fatal("manifest incomplete flag missing")
	}
}
