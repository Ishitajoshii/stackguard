package scout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSparsePartialReport(t *testing.T) {
	d := t.TempDir()
	p := Defaults()
	p.GitHub = "another-user"
	p.Resume = "C:/private/person.pdf"
	r := Research{Complete: false, Coverage: []string{"API query budget exhausted"}}
	if e := WriteReport(d, p, r, "test-session"); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"report.md", "recommendations.json", "manifest.json"} {
		b, e := os.ReadFile(filepath.Join(d, name))
		if e != nil || strings.Contains(string(b), p.Resume) {
			t.Fatal(name, e)
		}
	}
	if e := WriteReport(d, p, r, "again"); e == nil {
		t.Fatal("overwrote evidence")
	}
}
