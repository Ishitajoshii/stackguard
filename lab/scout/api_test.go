package scout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestBudgetAndNoRedirectCredentials(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("missing auth")
		}
		w.Header().Set("Location", "https://elsewhere.example")
		w.WriteHeader(302)
	}))
	defer server.Close()
	a := newAPI(server.URL, "test-only", "github", 1)
	var out any
	if e := a.Get(context.Background(), "/repos/a/b", nil, &out); e == nil {
		t.Fatal("redirect accepted")
	}
	if e := a.Get(context.Background(), "/repos/a/b", nil, &out); e == nil || calls != 1 {
		t.Fatal(e, calls)
	}
}
func TestNotionDoesNotFollowLinkedPages(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"results":[{"id":"11111111111111111111111111111111","type":"child_page","has_children":true,"child_page":{"title":"Do not follow"}},{"id":"22222222222222222222222222222222","type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Interest: backend"}]}}],"has_more":false}`))
	}))
	defer server.Close()
	a := newAPI(server.URL, "test", "notion", 5)
	notes, e := a.Notes(context.Background(), "00000000000000000000000000000000")
	if e != nil || calls != 1 || len(notes) != 1 {
		t.Fatal(notes, calls, e)
	}
}
