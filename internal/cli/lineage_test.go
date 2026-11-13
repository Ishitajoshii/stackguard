package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPrintLineage(t *testing.T) {
	// Output-format fixture only; this is not native-agent evidence.
	want := "identity\n  agent\n    tool [implicit]\n"
	path := filepath.Join(t.TempDir(), "lineage.txt")
	if err := os.WriteFile(path, []byte(want), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := printLineage(context.Background(), path, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

type lineageSink struct {
	*io.PipeWriter
	entered chan struct{}
	once    sync.Once
}

func (s *lineageSink) Write(p []byte) (int, error) {
	s.once.Do(func() { close(s.entered) })
	return s.PipeWriter.Write(p)
}

func TestPrintLineageCancellationUnblocksOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lineage.txt")
	if err := os.WriteFile(path, []byte("tree\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	out := &lineageSink{PipeWriter: w, entered: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- printLineage(ctx, path, out) }()
	select {
	case <-out.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("write did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not unblock output")
	}
}
