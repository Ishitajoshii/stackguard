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

// Algorithm fixtures exercise eligibility, not evidence of real AWS activity.
func TestUnusedIdentityKindsAlgorithmInputs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       string
		unresolved bool
		used       bool
		want       bool
	}{
		{"secret", "secret", false, false, true},
		{"credential set", "credential_set", false, false, true},
		{"role principal", "principal", false, false, true},
		{"observed role", "principal", false, true, false},
		{"unresolved role", "principal", true, false, false},
		{"identifier", "identifier", false, false, false},
		{"reference", "reference", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			id := "aws_role:arn:aws:iam::000000000000:role/algorithm-only"
			write := func(name string, values ...any) {
				t.Helper()
				f, err := os.Create(filepath.Join(d, name))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				for _, v := range values {
					if err := json.NewEncoder(f).Encode(v); err != nil {
						t.Fatal(err)
					}
				}
			}
			write("inventory.jsonl", model.Identity{ID: id, Kind: tc.kind, Unresolved: tc.unresolved, Bindings: []model.Binding{{Agent: "algorithm"}}})
			write("events.jsonl")
			if tc.used {
				write("edges.jsonl", model.Edge{NHI: id, Agent: "algorithm", Count: 1})
			} else {
				write("edges.jsonl")
			}
			if err := Risks(context.Background(), d, d, filepath.Join(d, "inventory.jsonl")); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(filepath.Join(d, "findings.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if !tc.want {
				if len(b) != 0 {
					t.Fatalf("unexpected finding: %s", b)
				}
				return
			}
			var finding model.Finding
			if err := json.Unmarshal(b, &finding); err != nil {
				t.Fatal(err)
			}
			if finding.Rule != "unused_in_window" || finding.NHI != id || finding.Severity != "low" || finding.Coverage == "" {
				t.Fatalf("incorrect absence finding: %+v", finding)
			}
		})
	}
}
