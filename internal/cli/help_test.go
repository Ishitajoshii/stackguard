package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpDoesNotRunCommandsOrEchoSuppliedValues(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"scan", "--help"}, {"proxy", "--help"}, {"trace", "--out", "private-test-value", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(context.Background(), args, nil, &stdout, &stderr); code != 0 {
			t.Fatal(args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage: nhi-probe") || strings.Contains(stdout.String(), "private-test-value") || stderr.Len() != 0 {
			t.Fatal(args, stdout.String(), stderr.String())
		}
	}
}
