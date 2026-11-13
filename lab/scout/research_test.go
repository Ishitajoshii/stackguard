package scout

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// Contract fixture, not a claim about a live provider or agent run.
func TestFinalRepositoryRecheckRejectsNewlyArchivedProject(t *testing.T) {
	finalChecks := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/repositories":
			w.Write([]byte(`{"items":[{"full_name":"org/backend","language":"Go","license":{"spdx_id":"MIT"}}]}`))
		case "/repos/org/backend/issues":
			json.NewEncoder(w).Encode([]map[string]any{{"number": 1, "state": "open", "body": strings.Repeat("clear acceptance ", 10)}})
		case "/repos/org/backend/community/profile":
			w.Write([]byte(`{}`))
		case "/repos/org/backend":
			finalChecks++
			w.Write([]byte(`{"full_name":"org/backend","archived":true,"license":{"spdx_id":"MIT"}}`))
		case "/repos/org/backend/issues/1":
			w.Write([]byte(`{"number":1,"state":"open"}`))
		default:
			w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()
	p := Defaults()
	p.GitHub = "example"
	p.Results = 1
	p.Interests = []string{"backend"}
	v, e := Discover(context.Background(), p, Summary{Reviewed: true}, newAPI(server.URL, "", "github", 160), false)
	if e != nil || finalChecks != 1 || len(v.Recommendations) != 0 {
		t.Fatal(v, e, finalChecks)
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
