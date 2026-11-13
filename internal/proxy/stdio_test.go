package proxy

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestChildProcess(t *testing.T) {
	if os.Getenv("STACKGUARD_TEST_CHILD") != "1" {
		return
	}
	os.Exit(0)
}
func TestUpstreamExitReleasesIdleInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	observerCtx, stop := context.WithCancel(ctx)
	o, _ := NewObserver(observerCtx, 1, "r", "a", "c", func(any) error { return nil })
	defer func() { stop(); o.Wait() }()
	reader, writer := io.Pipe()
	defer writer.Close()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	e = Stdio(ctx, StdioOptions{Command: exe, Args: []string{"-test.run=^TestChildProcess$"}, Env: append(os.Environ(), "STACKGUARD_TEST_CHILD=1", "GORACE=atexit_sleep_ms=0"), Input: reader, Output: &strings.Builder{}, Observer: o, Session: "s"})
	if e != nil || time.Since(start) > 2*time.Second {
		t.Fatal(e, time.Since(start))
	}
}
