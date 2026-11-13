package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputAndNoClobber(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	w, e := New(p)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Write(map[string]string{"value": "one\ntwo"}); e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	var got map[string]string
	if e = json.Unmarshal(b, &got); e != nil || got["value"] != "one\ntwo" {
		t.Fatalf("invalid record %q %v", b, e)
	}
	if _, e = New(p); e == nil {
		t.Fatal("clobbered")
	}
	m := filepath.Join(filepath.Dir(p), "manifest.json")
	if e = Publish(m, Manifest("test")); e != nil {
		t.Fatal(e)
	}
	if e = Publish(m, Manifest("test")); e == nil {
		t.Fatal("manifest clobbered")
	}
}
