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

func TestSingleSessionReplayPreservesCrossSessionParents(t *testing.T) {
	// Algorithm fixture, not an original agent capture. Repeated span IDs in
	// different source sessions must not collide after merging those sessions.
	d := t.TempDir()
	path := filepath.Join(d, "algorithm.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range []model.Event{
		{Version: 1, ID: "p", TS: time.Unix(1, 0), Source: "algorithm-test", Session: "a", Span: "same", RequestID: "req", Agent: "a", Kind: "delegation"},
		{Version: 1, ID: "c", TS: time.Unix(1, 0), Source: "algorithm-test", Session: "b", Span: "same", Parent: "same", ParentSession: "a", ParentRequestID: "req", Agent: "b", Kind: "tool_call"},
	} {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	manifest := filepath.Join(d, "original.json")
	if err := output.Publish(manifest, output.Manifest("algorithm-test-not-real-capture")); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(d, "out")
	if err := ReplayWithSessions(context.Background(), path, manifest, out, 4, "single"); err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(filepath.Join(out, "replay.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var events [4]model.Event
	for i := range events {
		if err := dec.Decode(&events[i]); err != nil {
			t.Fatal(err)
		}
		if err := events[i].Validate(); err != nil {
			t.Fatal(err)
		}
		if events[i].Session != events[0].Session {
			t.Fatal("sessions were not merged")
		}
	}
	for _, pair := range [][2]int{{0, 1}, {2, 3}} {
		p, c := events[pair[0]], events[pair[1]]
		if p.Span == c.Span || p.Span != c.Parent || p.RequestID != c.ParentRequestID || c.ParentSession != p.Session {
			t.Fatal("cross-session parent mapping lost or source spans collided")
		}
	}
	if events[0].Span == events[2].Span || events[0].ID == events[2].ID {
		t.Fatal("copies collided")
	}
	if err := ReplayWithSessions(context.Background(), path, manifest, filepath.Join(d, "bad"), 4, "unknown"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}
