package cloud

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestCredentialHelperHasNoMCPInput(t *testing.T) {
	if os.Getenv("STACKGUARD_TEST_HELPER") == "credential-stdin" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil || len(b) != 0 {
			os.Exit(3)
		}
		os.Stdout.WriteString("closed-input")
		os.Exit(0)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = previous }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := credentialCommand(ctx, os.Args[0], "-test.run=^TestCredentialHelperHasNoMCPInput$")
	cmd.Env = append(os.Environ(), "STACKGUARD_TEST_HELPER=credential-stdin")
	b, err := cmd.Output()
	if err != nil || string(b) != "closed-input" {
		t.Fatalf("credential helper read or waited on the open MCP pipe: %v", err)
	}
}
