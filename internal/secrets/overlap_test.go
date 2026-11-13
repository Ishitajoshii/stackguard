package secrets

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOverlappingCandidatesPreserveJSON(t *testing.T) {
	token := "sk-proj-" + strings.Repeat("a", 40)
	raw := `{"AZURE_OPENAI_API_KEY":"` + token + `","unrelated":"` + strings.Repeat("z", 100) + `"}`
	redacted := Redact(raw)
	if !json.Valid([]byte(redacted)) || strings.Contains(redacted, token) {
		t.Fatal(redacted)
	}
	if len(Detect(raw, "")) != 1 {
		t.Fatal("overlapping signatures double counted")
	}
}
