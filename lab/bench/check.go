package bench

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/Ishitajoshii/stackguard/internal/pipeline"
)

type ReplayCheck struct {
	SHA256   string `json:"sha256"`
	Bytes    int64  `json:"bytes"`
	Records  int64  `json:"records"`
	Sessions int64  `json:"distinct_sessions"`
}

type countedReader struct {
	io.Reader
	n int64
}

func (r *countedReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += int64(n)
	return n, err
}

// CheckReplay checks JSON syntax, hashes the exact bytes and counts sessions
// using an external sort rather than retaining a map of millions of sessions.
// It does not certify native provenance or credential-bearing correctness.
func CheckReplay(ctx context.Context, input, temp string) (result ReplayCheck, err error) {
	dir, err := os.MkdirTemp(temp, "stackguard-replay-check-*")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	f, err := os.Open(input)
	if err != nil {
		return result, err
	}
	defer f.Close()
	spool, err := os.Create(filepath.Join(dir, "sessions"))
	if err != nil {
		return result, err
	}
	defer spool.Close()
	w := bufio.NewWriterSize(spool, 64<<10)
	h := sha256.New()
	r := &countedReader{Reader: io.TeeReader(f, h)}
	err = pipeline.Lines(ctx, r, func(line string) error {
		var v struct {
			Session string `json:"session"`
		}
		if json.Unmarshal([]byte(line), &v) != nil || len(v.Session) != 64 {
			return errors.New("replay record must contain a rewritten SHA-256 session")
		}
		if _, e := hex.DecodeString(v.Session); e != nil {
			return errors.New("invalid replay session hash")
		}
		if _, e := w.WriteString(v.Session + "\n"); e != nil {
			return e
		}
		result.Records++
		return nil
	})
	if err != nil {
		return result, err
	}
	if err = errors.Join(w.Flush(), spool.Close()); err != nil {
		return result, err
	}
	result.Bytes, result.SHA256 = r.n, hex.EncodeToString(h.Sum(nil))
	sorted, err := pipeline.Sort(ctx, spool.Name(), dir)
	if err != nil {
		return result, err
	}
	sessions, err := os.Open(sorted)
	if err != nil {
		return result, err
	}
	defer sessions.Close()
	previous := ""
	err = pipeline.Lines(ctx, sessions, func(session string) error {
		if session != previous {
			result.Sessions++
			previous = session
		}
		return nil
	})
	return result, err
}
