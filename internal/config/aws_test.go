package config

import (
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"testing"
)

func TestAWSBundleOneCredential(t *testing.T) {
	d := Document{Agent: "lab", Servers: []Server{{Name: "tools", Env: map[string]string{"AWS_ACCESS_KEY_ID": "test-access", "AWS_SECRET_ACCESS_KEY": "test-secret", "AWS_SESSION_TOKEN": "test-session"}}}}
	ids := Resolve(d, "test", func(string) (string, bool) { return "", false })
	if len(ids) != 1 || ids[0].Kind != "credential_set" {
		t.Fatal(ids)
	}
	want, _ := secrets.AWSSet("test-access", "test-secret", "test-session")
	if ids[0].ID != want || len(ids[0].Bindings) != 3 {
		t.Fatal(ids)
	}
}
