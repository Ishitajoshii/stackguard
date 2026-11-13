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
