package config

import (
	"strings"
	"testing"
)

func TestCodexReferencesAndLiteral(t *testing.T) {
	token := "ghp_" + strings.Repeat("A", 36)
	d, e := (Codex{}).Parse([]byte("[mcp_servers.local]\ncommand='tool'\nargs=['--read']\nenv_vars=['MISSING']\n[mcp_servers.local.env]\nGITHUB_TOKEN='" + token + "'\n[mcp_servers.remote]\nurl='https://example.com/mcp'\nbearer_token_env_var='GITHUB_TOKEN'\n"))
	if e != nil {
		t.Fatal(e)
	}
	got := Resolve(d, "config.toml", func(s string) (string, bool) { return token, s == "GITHUB_TOKEN" })
	if len(got) != 3 {
		t.Fatal(got)
	}
	count := 0
	id := ""
	for _, v := range got {
		if v.Unresolved {
			count++
			continue
		}
		if id != "" && id != v.ID {
			t.Fatal("same token split")
		}
		id = v.ID
	}
	if count != 1 {
		t.Fatal("unresolved lost")
	}
	if len(d.Servers) != 2 || d.Servers[0].Command != "tool" {
		t.Fatal(d)
	}
}
func TestMalformedNoEcho(t *testing.T) {
	_, e := (Codex{}).Parse([]byte("secret = [ broken"))
	if e == nil || strings.Contains(e.Error(), "broken") {
		t.Fatal(e)
	}
	d, e := (AgentCore{}).Parse([]byte(`{"schema_version":1,"execution_role":"arn:aws:iam::000000000000:role/test"}`))
	if e != nil {
		t.Fatal(e)
	}
	ids := Resolve(d, "export.json", func(string) (string, bool) { t.Fatal("unexpected env lookup"); return "", false })
	if len(ids) != 1 || ids[0].Kind != "principal" {
		t.Fatal(ids)
	}
}
