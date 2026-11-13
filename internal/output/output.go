// Package output writes whole JSONL records and publishes completion manifests.
package output

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Ishitajoshii/stackguard/internal/model"
)

type Writer struct {
	mu      sync.Mutex
	file    *os.File
	Records uint64
}

// New refuses to overwrite evidence from an earlier run.
func New(path string) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, errors.New("create output directory failed")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("create output failed (destination may already exist)")
	}
	return &Writer{file: f}, nil
}
func (w *Writer) Write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return errors.New("encode record failed")
	}
	b = append(b, '\n')
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err = w.file.Write(b); err != nil {
		return errors.New("write record failed")
	}
	w.Records++
	return nil
}
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.file.Sync()
	return errors.Join(err, w.file.Close())
}
func Manifest(operation string) model.Manifest {
	var id [16]byte
	_, _ = rand.Read(id[:])
	return model.Manifest{Version: model.Version, Operation: operation, RunID: hex.EncodeToString(id[:]), Started: time.Now().UTC()}
}

// Publish writes in the destination directory then renames. Existing manifests
// are never silently replaced, including on Windows.
func Publish(path string, v any) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return errors.New("manifest destination already exists or is inaccessible")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".manifest-*")
	if err != nil {
		return errors.New("create manifest failed")
	}
	name := f.Name()
	defer os.Remove(name)
	if err = json.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func Checksum(path string) (model.Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return model.Source{}, errors.New("open source failed")
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return model.Source{Name: filepath.Base(path), SHA256: hex.EncodeToString(h.Sum(nil)), Bytes: n}, err
}
