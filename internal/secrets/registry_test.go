package secrets

import (
	"strings"
	"testing"
)

func TestEightProvidersAndContext(t *testing.T) {
	r := DefaultRegistry()
	if len(r.detectors) != 8 {
		t.Fatal(len(r.detectors))
	}
	cases := map[string]string{"github": "ghp_" + strings.Repeat("a", 36), "notion": "ntn_" + strings.Repeat("a", 32), "openai": "sk-proj-" + strings.Repeat("a", 40), "anthropic": "sk-ant-" + strings.Repeat("a", 40), "slack": "xoxb-" + strings.Repeat("1", 24), "google": "AIza" + strings.Repeat("A", 35), "aws": "AKIA" + strings.Repeat("A", 16), "azure": strings.Repeat("f", 32)}
	for name, value := range cases {
		matches := r.detectors[name].Detect(value, name+"_API_KEY")
		if len(matches) != 1 || matches[0].ID != Fingerprint(value) {
			t.Fatal(name, matches)
		}
	}
	value := strings.Repeat("f", 32)
	matches := Detect(`AZURE_OPENAI_API_KEY="`+value+`"`, "")
	if len(matches) != 1 || matches[0].Provider != "azure" {
		t.Fatal(matches)
	}
	if strings.Contains(Redact(`AZURE_OPENAI_API_KEY="`+value+`"`), value) {
		t.Fatal("contextual secret leaked")
	}
}
