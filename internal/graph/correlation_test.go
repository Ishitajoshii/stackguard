package graph

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitRequestCorrelation(t *testing.T) {
	d := t.TempDir()
	events := []model.Event{{Session: "local", Span: "proxyspan", RequestID: "hashed-id", Agent: "codex", Kind: "tool_call"}, {Session: "cloud", Span: "root", ParentSession: "local", ParentRequestID: "hashed-id", Agent: "cloud"}, {Session: "cloud", Span: "tool", Parent: "root", Agent: "cloud"}, {Session: "http", Span: "downstream", ParentSession: "cloud", Parent: "tool", Agent: "github"}}
	p := filepath.Join(d, "nodes")
	f, _ := os.Create(p)
	for _, e := range events {
		f.WriteString(NodeRecord(e))
	}
	f.Close()
	table, e := Index(context.Background(), p, d)
	if e != nil {
		t.Fatal(e)
	}
	defer table.Close()
	path, why, e := Path(context.Background(), table, events[3])
	if e != nil || why != "" || strings.Join(path, "/") != "codex/cloud/github" {
		t.Fatal(path, why, e)
	}
}
