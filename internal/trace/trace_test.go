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
	var previous string
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
	}
}
