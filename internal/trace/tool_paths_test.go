package trace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
)

func TestDistinctToolPathsDoNotMerge(t *testing.T) {
	// Algorithm-only records: native credential-bearing correctness is separate.
	d := t.TempDir()
	input := filepath.Join(d, "algorithm.jsonl")
	base := model.Event{Version: 1, TS: time.Unix(1, 0), Source: "algorithm-test", Session: "s", Agent: "coder", Kind: "tool_call", NHI: secrets.Fingerprint("algorithm identity"), Attribution: "implicit", Parent: "parent"}
	a, b := base, base
	a.ID, a.Span, a.Target = "a", "a", "mcp:github/issues"
	b.ID, b.Span, b.Target = "b", "b", "mcp:github/repos"
	parent := base
	parent.ID, parent.Span, parent.Parent, parent.Agent, parent.Target, parent.NHI = "parent", "parent", "", "supervisor", "delegate", ""
	completion := a
	completion.ID, completion.Kind, completion.Target, completion.NHI = "done", "tool_completion", "", ""
	f, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range []model.Event{completion, b, a, parent} {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var previous string
	for _, workers := range []int{1, 4, 8, 16} {
		out := filepath.Join(d, strings.Repeat("w", workers))
		m, err := Run(context.Background(), Options{Inputs: []Input{{"normalized", input}}, Output: out, Temp: d, Workers: workers, Lineage: true})
		if err != nil || !m.Complete {
			t.Fatal(m, err)
		}
		data, err := os.ReadFile(filepath.Join(out, "edges.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("merged distinct tools: %s", data)
		}
		seen := map[string]bool{}
		for _, line := range lines {
			var edge model.Edge
			if err := json.Unmarshal([]byte(line), &edge); err != nil {
				t.Fatal(err)
			}
			if len(edge.Path) != 4 || edge.Path[0] != "supervisor" || edge.Path[1] != "delegate" || edge.Path[2] != "coder" || edge.Count != 1 || edge.Unresolved != "" {
				t.Fatal(edge)
			}
			seen[edge.Path[3]] = true
		}
		if !seen[a.Target] || !seen[b.Target] {
			t.Fatal(seen)
		}
		if previous != "" && previous != string(data) {
			t.Fatal("worker-dependent tool paths")
		}
		previous = string(data)
	}
}
