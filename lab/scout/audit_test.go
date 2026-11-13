package scout

import (
	"context"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/correlation"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObservedCredentialAtRealHTTPBoundary(t *testing.T) {
	token := "ghp_" + strings.Repeat("E", 36)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("credential not transmitted")
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	a := newAPI(server.URL, token, "github", 2)
	a.AuditSession = "server-run"
	var events []model.Event
	a.Audit = func(v any) error { events = append(events, v.(model.Event)); return nil }
	ctx := correlation.With(context.Background(), correlation.Boundary{Session: "s", RequestID: "request"})
	var out any
	if e := a.Get(ctx, "/test", nil, &out); e != nil {
		t.Fatal(e)
	}
	if len(events) != 2 || events[0].NHI != secrets.Fingerprint(token) || events[0].ParentRequestID != "request" || events[1].Status != "succeeded" {
		t.Fatal(events)
	}
	b, _ := json.Marshal(events)
	if strings.Contains(string(b), token) {
		t.Fatal("secret in audit")
	}
}
