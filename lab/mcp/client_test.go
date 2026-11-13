package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestClientRealServerFileRoundTrip(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "data.txt"), []byte("real test file"), 0600)
	server := httptest.NewServer(Server{Tools: []Tool{{Name: "repository_file"}}, Call: FileCall(d)}.HTTP())
	defer server.Close()
	c, e := NewClient(server.URL, "s", nil)
	if e != nil {
		t.Fatal(e)
	}
	result, e := c.Call(context.Background(), "repository_file", json.RawMessage(`{"path":"data.txt"}`))
	if e != nil {
		t.Fatal(e)
	}
	if result.(map[string]any)["bytes_read"] != float64(14) {
		t.Fatal(result)
	}
}
