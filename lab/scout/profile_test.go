package scout

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuestionnaireGenericSaveEdit(t *testing.T) {
	p := Defaults()
	p.Unknown = map[string]any{"custom_label": "keep"}
	got, e := Questionnaire(strings.NewReader("another-user\nskip\ndatabases\nGo,Rust\ntiny\nskip\n3\nbackend\nyes\n"), &bytes.Buffer{}, p)
	if e != nil {
		t.Fatal(e)
	}
	if got.GitHub != "another-user" || strings.Join(got.Interests, ",") != "backend" || got.Resume != "" {
		t.Fatal(got)
	}
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if e = Save(path, got); e != nil {
		t.Fatal(e)
	}
	loaded, e := Load(path)
	if e != nil || loaded.Unknown["custom_label"] != "keep" {
		t.Fatal(loaded, e)
	}
	loaded.Effort = "stretch"
	if e = Save(path, loaded); e != nil {
		t.Fatal(e)
	}
}
func TestNoninteractiveAndNotionValidation(t *testing.T) {
	if _, e := Questionnaire(strings.NewReader(""), &bytes.Buffer{}, Defaults()); e == nil {
		t.Fatal("silent setup")
	}
	id, e := NotionID("https://app.notion.com/p/project-41763596c3dc80e5ac32cd606e2a7b9c?source=copy_link")
	if e != nil || len(id) != 32 {
		t.Fatal(id, e)
	}
	if _, e = NotionID("https://evil.example/41763596c3dc80e5ac32cd606e2a7b9c"); e == nil {
		t.Fatal("accepted foreign host")
	}
}
func TestGitHubProfileLinks(t *testing.T) {
	for _, input := range []string{"another-user", "https://github.com/another-user/"} {
		got, e := GitHubHandle(input)
		if e != nil || got != "another-user" {
			t.Fatal(got, e)
		}
	}
	for _, input := range []string{"https://evil.example/another-user", "https://github.com/org/repo", "https://user@github.com/profile"} {
		if _, e := GitHubHandle(input); e == nil {
			t.Fatal("invalid profile URL accepted")
		}
	}
}
