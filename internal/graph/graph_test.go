package graph

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are algorithm test records, not lab/native-capture evidence.
func TestLateParentsCyclesAndAmbiguity(t *testing.T) {
	ctx := context.Background()
	d := t.TempDir()
	p := filepath.Join(d, "nodes")
	events := []model.Event{{Session: "s", Span: "child", Parent: "parent", Agent: "tool"}, {Session: "s", Span: "parent", Agent: "researcher"}, {Session: "s", Span: "cycle", Parent: "cycle", Agent: "loop"}, {Session: "s", Span: "missing", Parent: "absent", Agent: "tool"}, {Session: "s", Span: "amb", Agent: "a"}, {Session: "s", Span: "amb", Agent: "b"}}
	f, _ := os.Create(p)
	for _, e := range events {
		f.WriteString(NodeRecord(e))
	}
	f.Close()
	table, e := Index(ctx, p, d)
	if e != nil {
		t.Fatal(e)
	}
	defer table.Close()
	path, why, e := Path(ctx, table, events[0])
	if e != nil || why != "" || strings.Join(path, "/") != "researcher/tool" {
		t.Fatal(path, why, e)
	}
	for _, i := range []int{2, 3, 4} {
		_, why, e = Path(ctx, table, events[i])
		if e != nil || why == "" {
			t.Fatal(i, why, e)
		}
	}
}

func TestAgentAndToolWithSameLabelRemainDistinct(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "nodes")
	e := model.Event{Session: "s", Span: "call", Agent: "read", Kind: "tool_call", Target: "read"}
	if err := os.WriteFile(p, []byte(NodeRecord(e)), 0600); err != nil {
		t.Fatal(err)
	}
	table, err := Index(context.Background(), p, d)
	if err != nil {
		t.Fatal(err)
	}
	defer table.Close()
	path, why, err := Path(context.Background(), table, e)
	if err != nil || why != "" || len(path) != 2 || path[0] != "read" || path[1] != "read" {
		t.Fatal("agent and tool are separate observed steps", path, why, err)
	}
}
