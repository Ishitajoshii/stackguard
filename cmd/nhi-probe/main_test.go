package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code int
	}{
		{"help", []string{"--help"}, 0},
		{"version", []string{"--version"}, 0},
		{"unimplemented command", []string{"trace"}, 2},
		{"extra args", []string{"--version", "unexpected"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out, err bytes.Buffer
			if got := run(tt.args, &out, &err); got != tt.code {
				t.Fatalf("exit=%d, want %d", got, tt.code)
			}
			if tt.code != 0 && (out.Len() != 0 || err.Len() == 0) {
				t.Fatal("failure must use stderr only")
			}
		})
	}
}

func TestInvalidArgumentsAreNotEchoed(t *testing.T) {
	const sensitive = "private-value-that-must-not-be-echoed"
	var out, err bytes.Buffer
	run([]string{sensitive}, &out, &err)
	if strings.Contains(out.String()+err.String(), sensitive) {
		t.Fatal("argument leaked into diagnostic output")
	}
}
