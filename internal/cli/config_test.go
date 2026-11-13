package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigReadBound(t *testing.T) {
	p := filepath.Join(t.TempDir(), "oversize.toml")
	if e := os.WriteFile(p, make([]byte, (1<<20)+1), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := boundedConfig(p); e == nil {
		t.Fatal("unbounded config accepted")
	}
}
