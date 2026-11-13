package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"strings"
	"sync"
	"testing"
)

func TestSessionIsolationAndSingleCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var events []model.Event
	o, e := NewObserver(ctx, 4, "run", "unknown", "bridge", func(v any) error { mu.Lock(); defer mu.Unlock(); events = append(events, v.(model.Event)); return nil })
	if e != nil {
		t.Fatal(e)
	}
	defer func() { cancel(); o.Wait() }()
	token := "ghp_" + strings.Repeat("D", 36)
	req := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"token":"` + token + `"}}}`)
	for _, s := range []string{"a", "b"} {
		if e = o.Observe(ctx, s, req, false, nil); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{"b", "a"} {
		if e = o.Observe(ctx, s, []byte(`{"id":1,"result":{"isError":true}}`), true, nil); e != nil {
			t.Fatal(e)
		}
	}
	if len(events) != 4 {
		t.Fatal(events)
	}
	for i, v := range events {
		if i < 2 && (v.Kind != "tool_call" || len(v.Credentials) != 1) {
			t.Fatal(v)
		}
		if i >= 2 && v.Status != "failed" {
			t.Fatal(v)
		}
		b, _ := json.Marshal(v)
		if bytes.Contains(b, []byte(token)) {
			t.Fatal("secret leak")
		}
	}
	if events[0].Span != events[3].Span || events[1].Span != events[2].Span {
		t.Fatal("cross-session response match")
	}
}
func TestExactFraming(t *testing.T) {
	raw := "{\"id\":1}\r\n{\"method\":\"notify\"}\n"
	var out bytes.Buffer
	if e := frames(context.Background(), strings.NewReader(raw), func(b []byte) error { out.Write(b); return nil }); e != nil || out.String() != raw {
		t.Fatal(out.String(), e)
	}
}
