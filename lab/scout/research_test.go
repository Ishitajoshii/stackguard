package scout

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQualification(t *testing.T) {
	var r Repository
	json.Unmarshal([]byte(`{"license":{"spdx_id":"MIT"}}`), &r)
	i := Issue{Number: 1, State: "open"}
	if !Suitable(r, i) {
		t.Fatal("valid issue rejected")
	}
	i.PR = json.RawMessage(`{"url":"test"}`)
	if Suitable(r, i) {
		t.Fatal("PR accepted as issue")
	}
	i.PR = nil
	r.Archived = true
	if Suitable(r, i) {
		t.Fatal("archived accepted")
	}
}
func TestFrozenScores(t *testing.T) {
	p := Defaults()
	p.GitHub = "example"
	r := Repository{Name: "org/backend", Language: "Go"}
	i := Issue{Number: 1, Body: "This issue has enough detail to explain the intended behavior and an independently verifiable acceptance criterion."}
	s := Summary{Skills: []SkillEvidence{{Skill: "Go", Kind: "self_reported_mention"}}}
	now := time.Unix(1, 0)
	a := Score(p, s, r, i, now, false, "")
	b := Score(p, s, r, i, now, false, "")
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if string(aa) != string(bb) || a.Components["skill_fit"] != 30 || a.Components["interest_fit"] != 25 {
		t.Fatal(a, b)
	}
	p.Interests = []string{"astronomy"}
	c := Score(p, s, r, i, now, false, "")
	if c.Score >= a.Score {
		t.Fatal("interest change ignored")
	}
}
