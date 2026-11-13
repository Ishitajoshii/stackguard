package pipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransformFailureCancelsBlockedProducer(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "input")
	os.WriteFile(p, []byte(strings.Repeat("a\n", 10000)), 0600)
	e := Transform(context.Background(), p, 4, func(s string) string { return s }, func(string) ([]string, error) { return nil, errors.New("test failure") }, &bytes.Buffer{})
	if e == nil {
		t.Fatal("swallowed error")
	}
}
