package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalSort(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "in")
	f, _ := os.Create(p)
	for i := 20000; i >= 0; i-- {
		fmt.Fprintf(f, "%06d%s\n", i, strings.Repeat("x", 500))
	}
	f.Close()
	out, e := Sort(context.Background(), p, d)
	if e != nil {
		t.Fatal(e)
	}
	r, _ := os.Open(out)
	defer r.Close()
	prev := ""
	count := 0
	e = Lines(context.Background(), r, func(s string) error {
		if s < prev {
			return errors.New("unsorted")
		}
		prev = s
		count++
		return nil
	})
	if e != nil || count != 20001 {
		t.Fatal(e, count)
	}
}
func TestCancelAndOversized(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := Lines(ctx, strings.NewReader("x\n"), func(string) error { return nil }); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := Lines(context.Background(), strings.NewReader(strings.Repeat("x", MaxRecord+1)), func(string) error { return nil }); e == nil {
		t.Fatal("unbounded record")
	}
}
