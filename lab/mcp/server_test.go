package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRealFileToolAndRequestID(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "README.md"), []byte("actual filesystem bytes"), 0600)
	s := Server{Tools: []Tool{{Name: "repository_file"}}, Call: FileCall(d)}
	r, e := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":"exact-id","method":"tools/call","params":{"name":"repository_file","arguments":{"path":"README.md"}}}`))
	if e != nil || string(r.ID) != `"exact-id"` {
		t.Fatal(r, e)
	}
	b, _ := json.Marshal(r)
	if len(b) == 0 {
		t.Fatal("empty response")
	}
	if _, e = s.Call(context.Background(), "repository_file", json.RawMessage(`{"path":"../outside"}`)); e == nil {
		t.Fatal("escaped root")
	}
	if r, e = s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)); e != nil || r != nil {
		t.Fatal(r, e)
	}
}
