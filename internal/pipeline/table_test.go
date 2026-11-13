package pipeline

import (
	"context"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskJoin(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "input")
	f, _ := os.Create(p)
	for i := 100; i >= 0; i-- {
		fmt.Fprintf(f, "%s\tvalue-%d\n", secrets.Fingerprint(fmt.Sprint(i)), i)
	}
	f.Close()
	table, e := BuildTable(context.Background(), p, d)
	if e != nil {
		t.Fatal(e)
	}
	defer table.Close()
	for i := 0; i <= 100; i++ {
		b, ok, e := table.Get(secrets.Fingerprint(fmt.Sprint(i)))
		if e != nil || !ok || string(b) != fmt.Sprintf("value-%d", i) {
			t.Fatal(i, string(b), ok, e)
		}
	}
	if _, ok, e := table.Get(secrets.Fingerprint("missing")); e != nil || ok {
		t.Fatal(ok, e)
	}
}
