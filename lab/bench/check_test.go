package bench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckReplayCountsRealCaptureCopies(t *testing.T) {
	d := t.TempDir()
	base := filepath.Join("..", "..", "captures", "codex-file-http-20261114", "proxy")
	for _, mode := range []string{"copies", "single"} {
		out := filepath.Join(d, mode)
		if err := ReplayWithSessions(context.Background(), filepath.Join(base, "proxy.jsonl"), filepath.Join(base, "manifest.json"), out, 6, mode); err != nil {
			t.Fatal(err)
		}
		input := filepath.Join(out, "replay.jsonl")
		result, err := CheckReplay(context.Background(), input, d)
		if err != nil {
			t.Fatal(err)
		}
		want := int64(3)
		if mode == "single" {
			want = 1
		}
		b, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		if result.Sessions != want || result.Records != 6 || result.Bytes != int64(len(b)) || result.SHA256 != hex.EncodeToString(h[:]) {
			t.Fatal(mode, result)
		}
	}
}
