package proxy

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHTTPStreamingPreservesWire(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":"a","method":"tools/call","params":{"name":"read"}}`
	response := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":\"a\",\"result\":{}}\n\n"
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if string(b) != request || r.Header.Get("Mcp-Session-Id") != "session" {
			t.Error("request mutated")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Mcp-Session-Id", "session")
		io.WriteString(w, response)
		w.(http.Flusher).Flush()
	}))
	defer up.Close()
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var got []model.Event
	o, _ := NewObserver(ctx, 2, "r", "unknown", "http", func(v any) error { mu.Lock(); defer mu.Unlock(); got = append(got, v.(model.Event)); return nil })
	defer func() { cancel(); o.Wait() }()
	h, e := HTTP(up.URL, o)
	defer h.(interface{ CloseIdleConnections() }).CloseIdleConnections()
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	r, _ := http.NewRequest("POST", server.URL, strings.NewReader(request))
	r.Header.Set("Mcp-Session-Id", "session")
	res, e := http.DefaultClient.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(res.Body)
	res.Body.Close()
	if e != nil || string(b) != response || res.Header.Get("Mcp-Session-Id") != "session" {
		t.Fatal(string(b), e)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[1].Status != "succeeded" {
		t.Fatal(got)
	}
}
func TestRejectOrigin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	o, _ := NewObserver(ctx, 1, "r", "a", "c", func(any) error { return nil })
	defer func() { cancel(); o.Wait() }()
	h, _ := HTTP("http://127.0.0.1:1", o)
	r := httptest.NewRequest("GET", "http://localhost:9000/mcp", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
