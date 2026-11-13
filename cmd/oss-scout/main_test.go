package main

import (
	"bytes"
	"context"
	"github.com/Ishitajoshii/stackguard/lab/scout"
	"os"
	"path/filepath"
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
