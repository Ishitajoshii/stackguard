package output

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

type shortFile struct {
	*os.File
	short bool
}

func (f *shortFile) Write(b []byte) (int, error) {
	if f.short {
		n, _ := f.File.Write(b[:len(b)/2])
		return n, io.ErrShortWrite
	}
	return f.File.Write(b)
}
func TestPartialRecordRollback(t *testing.T) {
	p := filepath.Join(t.TempDir(), "records.jsonl")
	f, e := os.Create(p)
	if e != nil {
		t.Fatal(e)
	}
	injected := &shortFile{File: f}
	w := &Writer{file: injected}
	if e = w.Write(map[string]int{"first": 1}); e != nil {
		t.Fatal(e)
	}
	injected.short = true
	if w.Write(map[string]int{"second": 2}) == nil {
		t.Fatal("partial write accepted")
	}
	if w.Write(map[string]int{"third": 3}) == nil {
		t.Fatal("failed writer reused")
	}
	w.Close()
	b, e := os.ReadFile(p)
	if e != nil || string(b) != "{\"first\":1}\n" || w.Records != 1 {
		t.Fatal(string(b), e, w.Records)
	}
}
