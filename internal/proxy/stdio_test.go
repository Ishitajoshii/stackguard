package proxy

import (
	"bufio"
	"context"
	"github.com/Ishitajoshii/stackguard/internal/model"
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
	if os.Getenv("STACKGUARD_TEST_MODE") == "protocol" {
		reader := bufio.NewScanner(os.Stdin)
		for _, exchange := range protocolExchanges {
			if !reader.Scan() || reader.Text() != exchange.request {
				os.Exit(3)
			}
			if exchange.response != "" {
				if _, err := io.WriteString(os.Stdout, exchange.response+"\n"); err != nil {
					os.Exit(4)
				}
			}
		}
		os.Exit(0)
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

// These are protocol fixtures, not agent behavior. The child validates every
// received byte so control messages cannot disappear unnoticed in the proxy.
var protocolExchanges = []struct{ request, response string }{
	{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}}`},
	{`{"jsonrpc":"2.0","method":"notifications/initialized"}`, ""},
	{`{"jsonrpc":"2.0","id":"list","method":"tools/list"}`, `{"jsonrpc":"2.0","id":"list","result":{"tools":[]}}`},
	{`{"jsonrpc":"2.0","id":"ok","method":"tools/call","params":{"name":"read","arguments":{}}}`, `{"jsonrpc":"2.0","id":"ok","result":{"content":[{"type":"text","text":"ok"}]}}`},
	{`{"jsonrpc":"2.0","id":"cancel","method":"tools/call","params":{"name":"wait","arguments":{}}}`, `{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"p","progress":1}}`},
	{`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":"cancel","reason":"test"}}`, `{"jsonrpc":"2.0","id":"cancel","error":{"code":-32800,"message":"cancelled"}}`},
	{`{"jsonrpc":"2.0","id":9,"method":"ping"}`, `{"jsonrpc":"2.0","id":9,"result":{}}`},
}

func TestStdioForwardsFullProtocolExchange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var events []model.Event
	var mu sync.Mutex
	o, err := NewObserver(ctx, 4, "r", "a", "c", func(v any) error {
		mu.Lock()
		defer mu.Unlock()
		if e, ok := v.(model.Event); ok {
			events = append(events, e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var input, want, got strings.Builder
	for _, exchange := range protocolExchanges {
		input.WriteString(exchange.request + "\n")
		if exchange.response != "" {
			want.WriteString(exchange.response + "\n")
		}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	err = Stdio(ctx, StdioOptions{Command: exe, Args: []string{"-test.run=^TestChildProcess$"}, Env: append(os.Environ(), "STACKGUARD_TEST_CHILD=1", "STACKGUARD_TEST_MODE=protocol", "GORACE=atexit_sleep_ms=0"), Input: io.NopCloser(strings.NewReader(input.String())), Output: &got, Observer: o, Session: "s"})
	cancel()
	o.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != want.String() {
		t.Fatal("protocol response bytes changed or disappeared")
	}
	counts := map[string]int{}
	for _, event := range events {
		counts[event.Kind]++
	}
	if counts["tool_call"] != 2 || counts["tool_completion"] != 2 {
		t.Fatalf("wrong observed call/result counts: %v", counts)
	}
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
