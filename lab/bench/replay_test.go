package bench

import (
	"context"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplayRewritesRelationships(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "algorithm.jsonl")
	f, _ := os.Create(path)
	enc := json.NewEncoder(f)
	enc.Encode(model.Event{Version: 1, ID: "p", TS: time.Unix(1, 0), Source: "algorithm-test", Session: "s", Span: "p", Agent: "a", Kind: "delegation"})
	enc.Encode(model.Event{Version: 1, ID: "c", TS: time.Unix(1, 0), Source: "algorithm-test", Session: "s", Span: "c", Parent: "p", Agent: "b", Kind: "tool_call"})
	f.Close()
	manifest := filepath.Join(d, "original.json")
	output.Publish(manifest, output.Manifest("algorithm-test-not-real-capture"))
	out := filepath.Join(d, "out")
	if e := Replay(context.Background(), path, manifest, out, 4); e != nil {
		t.Fatal(e)
	}
	f, _ = os.Open(filepath.Join(out, "replay.jsonl"))
	defer f.Close()
	dec := json.NewDecoder(f)
	var e [4]model.Event
	for i := range e {
		dec.Decode(&e[i])
	}
	if e[0].Span != e[1].Parent || e[2].Span != e[3].Parent || e[0].Session == e[2].Session {
		t.Fatal("broken mapping")
	}
}
