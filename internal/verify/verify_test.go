package verify

import (
	"context"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"net/http"
	"testing"
)

type forbiddenTransport struct{ t *testing.T }

func (f forbiddenTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.t.Fatal("non-owned credential sent to provider")
	return nil, nil
}
func TestMismatchNeverSends(t *testing.T) {
	client := &http.Client{Transport: forbiddenTransport{t}}
	r := Check(context.Background(), Owned{ID: secrets.Fingerprint("owned"), Provider: "github", Env: "DEDICATED"}, func(string) (string, bool) { return "different", true }, client)
	if r.Status != "skipped" {
		t.Fatal(r)
	}
}
