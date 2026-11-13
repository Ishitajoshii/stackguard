package scout

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalSummaryOmitsPrivateText(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "resume with spaces.txt")
	os.WriteFile(path, []byte("Private Name private@example.com\nPython and Go projects. Google is not Go by itself."), 0600)
	p := Defaults()
	p.Resume = path
	s, e := ExtractSummary(context.Background(), p, "", "", d)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), "private@example.com") || strings.Contains(string(b), "Private Name") || s.Reviewed {
		t.Fatal(string(b))
	}
	if len(s.Skills) != 2 {
		t.Fatal(s.Skills)
	}
	p.Resume = ""
	s, e = ExtractSummary(context.Background(), p, "", "", d)
	if e != nil || len(s.Coverage) < 2 {
		t.Fatal(s, e)
	}
}
