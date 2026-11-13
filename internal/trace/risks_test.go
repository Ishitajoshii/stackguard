package trace

import (
	"context"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRiskSemanticsAlgorithmInputs(t *testing.T) {
	d := t.TempDir()
	id := secrets.Fingerprint("algorithm-only")
	f, _ := os.Create(filepath.Join(d, "edges.jsonl"))
	for _, a := range []string{"a", "b", "c"} {
		json.NewEncoder(f).Encode(model.Edge{NHI: id, Agent: a, Count: 1})
	}
	f.Close()
	f, _ = os.Create(filepath.Join(d, "events.jsonl"))
	json.NewEncoder(f).Encode(model.Event{NHI: id, Agent: "b", DelegatedFrom: "a", Attribution: "implicit", Evidence: []string{"algorithm"}})
	f.Close()
	inv := filepath.Join(d, "inventory.jsonl")
	f, _ = os.Create(inv)
	json.NewEncoder(f).Encode(model.Identity{ID: secrets.Fingerprint("unused"), Kind: "secret"})
	f.Close()
	if e := Risks(context.Background(), d, d, inv); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(d, "findings.jsonl"))
	for _, rule := range []string{"shared_identity", "unused_in_window", "unwired_observation"} {
		if !strings.Contains(string(b), rule) {
			t.Fatal(rule, string(b))
		}
	}
	if strings.Contains(string(b), "token_passthrough") {
		t.Fatal("implicit delegation is not token transfer")
	}
}
