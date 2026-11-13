package scan

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkBoundary(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "input")
	token := "ghp_" + strings.Repeat("B", 36)
	prefix := strings.Repeat(".", 65530)
	os.WriteFile(p, []byte(prefix+token+strings.Repeat(".", 70000)), 0600)
	count := 0
	n, e := File(context.Background(), p, func(off int64, m secrets.Match) error {
		count++
		if off != 65530 || m.ID != secrets.Fingerprint(token) {
			t.Fatal(off, m)
		}
		return nil
	})
	if e != nil || count != 1 || n != int64(len(prefix)+len(token)+70000) {
		t.Fatal(n, count, e)
	}
}
func TestDeterminismAndDedup(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "input")
	os.Mkdir(root, 0700)
	token := "ghp_" + strings.Repeat("C", 36)
	os.WriteFile(filepath.Join(root, "a"), []byte(token), 0600)
	os.WriteFile(filepath.Join(root, "b"), []byte(token), 0600)
	var previous string
	for _, workers := range []int{1, 4, 8, 16} {
		out := filepath.Join(d, strings.Repeat("w", workers))
		m, e := Run(context.Background(), Options{Roots: []string{root}, Output: out, Temp: d, Workers: workers})
		if e != nil || !m.Complete || m.Records != 1 || m.Bytes != 80 {
			t.Fatal(m, e)
		}
		b, _ := os.ReadFile(filepath.Join(out, "inventory.jsonl"))
		if strings.Contains(string(b), token) {
			t.Fatal("secret leaked")
		}
		if previous != "" && previous != string(b) {
			t.Fatal("nondeterministic")
		}
		previous = string(b)
	}
}
