package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/Ishitajoshii/stackguard/lab/scout"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorAndNoninteractive(t *testing.T) {
	d := t.TempDir()
	in, e := os.Open(os.DevNull)
	if e != nil {
		t.Fatal(e)
	}
	defer in.Close()
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"init", "--profile", filepath.Join(d, "missing.yaml")}, in, &out, &errout); code != 2 {
		t.Fatal(code)
	}
	p := scout.Defaults()
	p.GitHub = "another-user"
	path := filepath.Join(d, "profile.yaml")
	scout.Save(path, p)
	out.Reset()
	errout.Reset()
	if code := run(context.Background(), []string{"doctor", "--profile", path, "--interests", "databases", "--hours", "8"}, in, &out, &errout); code != 0 {
		t.Fatal(code, errout.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(`"profile_valid":true`)) {
		t.Fatal(out.String())
	}
}

func TestServeStdioReadsRepositoryFile(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	// Protocol fixture against a real repository file, not agent evidence.
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"repository_file","arguments":{"path":"README.md"}}}` + "\n"
	path := filepath.Join(t.TempDir(), "input.jsonl")
	if err := os.WriteFile(path, []byte(request), 0600); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	var out, stderr bytes.Buffer
	if code := run(context.Background(), []string{"serve", "--stdio", "--files", root}, in, &out, &stderr); code != 0 {
		t.Fatal(code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), hex.EncodeToString(sum[:])) || strings.Contains(out.String(), `"isError":true`) {
		t.Fatal("stdio response did not contain the repository file hash", out.String())
	}
}
