package scan

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
)

func TestInterruptRetainsAlreadyObservedCandidates(t *testing.T) {
	d := t.TempDir()
	input := filepath.Join(d, "algorithm-fixture")
	f, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	// Sparse regular-file tail keeps scanning active after the first match.
	token := "ghp_" + strings.Repeat("q", 30)
	if _, err = f.WriteString(token + "\n"); err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(64 << 20); err != nil {
		t.Fatal(err)
	}
	f.Close()
	out := filepath.Join(d, "out")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Run(ctx, Options{Roots: []string{input}, Output: out, Temp: d, Workers: 1})
		done <- err
	}()
	path := filepath.Join(out, "observations.jsonl")
	var first []byte
	for len(first) == 0 && ctx.Err() == nil {
		if f, err := os.Open(path); err == nil {
			line, readErr := bufio.NewReader(f).ReadBytes('\n')
			f.Close()
			if readErr == nil {
				first = line
			}
		}
		if len(first) == 0 {
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	if err = <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("scan did not interrupt: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("no candidate was published while scan was active")
	}
	var identity model.Identity
	if json.Unmarshal(first, &identity) != nil || identity.ID != secrets.Fingerprint(token) || strings.Contains(string(first), token) {
		t.Fatal("candidate missing or raw value exposed")
	}
	f, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	if err = decoder.Decode(&identity); err != nil {
		t.Fatal("published candidate was lost during cleanup", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest model.Manifest
	if json.Unmarshal(b, &manifest) != nil || manifest.Complete {
		t.Fatal("interrupted scan marked complete")
	}
	if !strings.Contains(strings.Join(manifest.Notes, "\n"), "workspace retained for explicit cleanup") {
		t.Fatal("interrupted workspace cleanup was not recorded")
	}
}
