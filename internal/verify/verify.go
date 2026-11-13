// Package verify never uses values found in public files. Only a separately
// allowlisted environment value with the exact full fingerprint can be checked.
package verify

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Owned struct {
	ID       string `json:"fingerprint"`
	Provider string `json:"provider"`
	Env      string `json:"env"`
}
type Result struct {
	ID       string    `json:"fingerprint"`
	Provider string    `json:"provider"`
	Status   string    `json:"status"`
	Reason   string    `json:"reason"`
	Checked  time.Time `json:"checked_at"`
}

func Check(ctx context.Context, entry Owned, env func(string) (string, bool), client *http.Client) Result {
	r := Result{ID: entry.ID, Provider: entry.Provider, Status: "skipped", Checked: time.Now().UTC()}
	token, ok := env(entry.Env)
	if !ok || !model.ValidIdentity(entry.ID) || secrets.Fingerprint(token) != entry.ID {
		r.Reason = "owned environment value missing or fingerprint mismatch"
		return r
	}
	endpoint := ""
	switch entry.Provider {
	case "github":
		endpoint = "https://api.github.com/user"
	case "notion":
		endpoint = "https://api.notion.com/v1/users/me"
	default:
		r.Reason = "no universal verifier for this configured credential type; use the documented provider-specific read-only procedure"
		return r
	}
	request, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "stackguard-owned-verification")
	if entry.Provider == "notion" {
		request.Header.Set("Notion-Version", "2025-09-03")
	} else {
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	response, e := client.Do(request)
	if e != nil {
		r.Status = "unknown"
		r.Reason = "provider unavailable or request cancelled"
		return r
	}
	defer response.Body.Close()
	io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	switch response.StatusCode {
	case 200:
		r.Status = "accepted"
		r.Reason = "provider accepted this read-only identity request; no broader permissions inferred"
	case 401:
		r.Status = "rejected"
		r.Reason = "provider rejected authentication"
	default:
		r.Status = "unknown"
		r.Reason = "read operation failed; scope, rate limit or provider state may explain it"
	}
	return r
}
func Run(ctx context.Context, inventory, allowlist, dir string) error {
	f, e := os.Open(allowlist)
	if e != nil {
		return errors.New("owned credential allowlist unavailable")
	}
	b, e := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	f.Close()
	if e != nil || len(b) > 64<<10 {
		return errors.New("allowlist exceeds limit")
	}
	var entries []Owned
	if json.Unmarshal(b, &entries) != nil || len(entries) > 64 {
		return errors.New("invalid owned credential allowlist")
	}
	owned := map[string]Owned{}
	for _, entry := range entries {
		if !model.ValidIdentity(entry.ID) || entry.Env == "" {
			return errors.New("allowlist requires full fingerprints and environment references")
		}
		owned[entry.ID] = entry
	}
	f, e = os.Open(inventory)
	if e != nil {
		return e
	}
	defer f.Close()
	w, e := output.New(filepath.Join(dir, "verification.jsonl"))
	if e != nil {
		return e
	}
	defer w.Close()
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return pipeline.Lines(ctx, f, func(line string) error {
		var id model.Identity
		if json.Unmarshal([]byte(line), &id) != nil {
			return errors.New("invalid inventory")
		}
		entry, ok := owned[id.ID]
		if !ok || id.Unresolved {
			return nil
		}
		delete(owned, id.ID)
		return w.Write(Check(ctx, entry, os.LookupEnv, client))
	})
}
