package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestStdioCancellationDoesNotBlockControlMessages(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	server := Server{Tools: []Tool{{Name: "wait"}}, Call: func(ctx context.Context, _ string, _ json.RawMessage) (any, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}}
	input, writer := io.Pipe()
	defer writer.Close()
	var output bytes.Buffer
	ctx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Stdio(ctx, input, &output) }()
	io.WriteString(writer, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"wait\"}}\n")
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("tool never started")
	}
	io.WriteString(writer, "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/cancelled\",\"params\":{\"requestId\":1}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"ping\"}\n")
	select {
	case <-cancelled:
	case <-ctx.Done():
		t.Fatal("cancellation was blocked behind tool")
	}
	writer.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("server did not finish")
	}
	decoder := json.NewDecoder(&output)
	seen := map[int]bool{}
	for decoder.More() {
		var response struct {
			ID int `json:"id"`
		}
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		seen[response.ID] = true
	}
	if !seen[1] || !seen[2] {
		t.Fatal("missing cancelled response or ping")
	}
}

func TestStdioInterruptClosesIdleInput(t *testing.T) {
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- (Server{}).Stdio(ctx, input, io.Discard) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("interrupt hidden")
		}
	case <-time.After(time.Second):
		t.Fatal("idle stdin blocked shutdown")
	}
}
