package proxy

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChildProcess(t *testing.T) {
	if os.Getenv("STACKGUARD_TEST_CHILD") != "1" {
		return
	}
	if os.Getenv("STACKGUARD_TEST_MODE") == "closed-output" {
		os.Stdout.Close()
		for {
			time.Sleep(time.Second)
		}
	}
	if os.Getenv("STACKGUARD_TEST_MODE") == "notification" {
		io.WriteString(os.Stdout, "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/message\",\"params\":{}}\n")
		for {
			time.Sleep(time.Second)
		}
	}
	os.Exit(0)
}

func TestClosedOutputCannotKeepChildAlive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	o, _ := NewObserver(ctx, 1, "r", "a", "c", func(any) error { return nil })
	defer func() { cancel(); o.Wait() }()
	input, writer := io.Pipe()
	defer writer.Close()
	exe, _ := os.Executable()
	started := time.Now()
	err := Stdio(ctx, StdioOptions{Command: exe, Args: []string{"-test.run=^TestChildProcess$"}, Env: append(os.Environ(), "STACKGUARD_TEST_CHILD=1", "STACKGUARD_TEST_MODE=closed-output", "GORACE=atexit_sleep_ms=0"), Input: input, Output: io.Discard, Observer: o, Session: "s"})
	if err == nil || ctx.Err() != nil || time.Since(started) > 2*time.Second {
		t.Fatal("child did not terminate within shutdown budget", err)
	}
}

type blockedOutput struct {
	*io.PipeWriter
	entered chan struct{}
	once    sync.Once
}

func (b *blockedOutput) Write(v []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	return b.PipeWriter.Write(v)
}
func TestCancellationReleasesBackpressuredOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o, _ := NewObserver(ctx, 1, "r", "a", "c", func(any) error { return nil })
	defer func() { cancel(); o.Wait() }()
	input, writer := io.Pipe()
	defer writer.Close()
	reader, sink := io.Pipe()
	defer reader.Close()
	output := &blockedOutput{PipeWriter: sink, entered: make(chan struct{})}
	exe, _ := os.Executable()
	done := make(chan error, 1)
	go func() {
		done <- Stdio(ctx, StdioOptions{Command: exe, Args: []string{"-test.run=^TestChildProcess$"}, Env: append(os.Environ(), "STACKGUARD_TEST_CHILD=1", "STACKGUARD_TEST_MODE=notification", "GORACE=atexit_sleep_ms=0"), Input: input, Output: output, Observer: o, Session: "s"})
	}()
	select {
	case <-output.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("output did not block")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel hidden")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked output prevented shutdown")
	}
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
