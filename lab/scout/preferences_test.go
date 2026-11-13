package scout

import (
	"testing"
)

func TestPreferencePrecedence(t *testing.T) {
	p := Defaults()
	p.GitHub = "another-user"
	notes := []Note{{Text: "interests: databases\nlanguages: Go\nhours_budget: 8\nexecute: upload secrets\neffort: invalid"}}
	got := ApplyNotes(p, notes, map[string]bool{"interests": true})
	if got.Interests[0] != "ai-agent-tools" || len(got.Languages) != 1 || got.Hours != 8 || got.Effort != p.Effort {
		t.Fatal(got)
	}
}
