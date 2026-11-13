package proxy

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPDisconnectReleasesPendingID(t *testing.T) {
	entered := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	defer upstream.Close()
	observerCtx, stop := context.WithCancel(context.Background())
	events := make(chan model.Event, 8)
	observer, _ := NewObserver(observerCtx, 1, "run", "a", "c", func(v any) error { events <- v.(model.Event); return nil })
	defer func() { stop(); observer.Wait() }()
	handler, _ := HTTP(upstream.URL, observer)
	defer handler.(interface{ CloseIdleConnections() }).CloseIdleConnections()
	server := httptest.NewServer(handler)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "POST", server.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read"}}`))
	request.Header.Set("X-Stackguard-Session", "session")
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		r, e := http.DefaultClient.Do(request)
		if e == nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream not reached")
	}
	cancel()
	<-returned
	select {
	case e := <-events:
		if e.Kind != "tool_call" {
			t.Fatal(e.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("missing call")
	}
	select {
	case e := <-events:
		if e.Kind != "tool_completion" || e.Status != "failed" {
			t.Fatal(e.Kind, e.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled ID not completed")
	}
	if e := observer.Observe(context.Background(), "session", []byte(`{"id":1,"method":"tools/call","params":{"name":"retry"}}`), false, nil); e != nil {
		t.Fatal("pending ID leaked", e)
	}
}
