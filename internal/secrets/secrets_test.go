package secrets

import (
	"strings"
	"testing"
)

func TestFingerprintsAndMask(t *testing.T) {
	if Fingerprint("abc") != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatal("not SHA256")
	}
	if Mask("short") == "short" {
		t.Fatal("short secret exposed")
	}
	token := "ghp_" + strings.Repeat("A", 36)
	m := Detect(token, "")
	if len(m) != 1 || m[0].Provider != "github" {
		t.Fatal("signature")
	}
	if strings.Contains(Redact("prefix "+token+" suffix"), token) {
		t.Fatal("leak")
	}
}
func TestStructuredRedaction(t *testing.T) {
	v := JSON(map[string]any{"Authorization": "Bearer private", "nested": []any{map[string]any{"password": "hidden"}}}, "").(map[string]any)
	if v["Authorization"] != "[redacted]" {
		t.Fatal("header leak")
	}
	if Detect("not-a-secret", "") != nil {
		t.Fatal("unlabelled text")
	}
	if len(Detect(strings.Repeat("f", 32), "AZURE_API_KEY")) != 1 {
		t.Fatal("context key")
	}
}
func FuzzRedaction(f *testing.F) {
	f.Add("hello")
	f.Fuzz(func(t *testing.T, s string) {
		_ = Redact(s)
		if len(Fingerprint(s)) != 64 {
			t.Fatal("digest")
		}
	})
}
