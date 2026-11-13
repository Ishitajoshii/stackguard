package pipeline

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

// Table is an immutable disk index. Keys are full 64-character SHA-256 hashes.
// Both index and values stay on disk, including for one enormous session.
type Table struct {
	index, data *os.File
	count       int64
}

func BuildTable(ctx context.Context, input, dir string) (*Table, error) {
	p, e := Sort(ctx, input, dir)
	if e != nil {
		return nil, e
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	idx, e := os.CreateTemp(dir, "index-*")
	if e != nil {
		return nil, e
	}
	data, e := os.CreateTemp(dir, "values-*")
	if e != nil {
		idx.Close()
		return nil, e
	}
	t := &Table{index: idx, data: data}
	indexWriter := bufio.NewWriterSize(idx, 64<<10)
	dataWriter := bufio.NewWriterSize(data, 64<<10)
	var offset int64
	previous := ""
	previousValue := ""
	e = Lines(ctx, f, func(line string) error {
		k, v, ok := strings.Cut(line, "\t")
		if !ok || len(k) != 64 {
			return errors.New("invalid table key")
		}
		if k == previous {
			if v != previousValue {
				return errors.New("ambiguous disk-table key")
			}
			return nil
		}
		previous = k
		previousValue = v
		var record [76]byte
		copy(record[:64], k)
		binary.LittleEndian.PutUint64(record[64:72], uint64(offset))
		binary.LittleEndian.PutUint32(record[72:], uint32(len(v)))
		if _, e := indexWriter.Write(record[:]); e != nil {
			return e
		}
		if _, e := io.WriteString(dataWriter, v); e != nil {
			return e
		}
		offset += int64(len(v))
		t.count++
		return nil
	})
	if e == nil {
		e = errors.Join(indexWriter.Flush(), dataWriter.Flush())
	}
	if e != nil {
		t.Close()
		return nil, e
	}
	return t, nil
}
func (t *Table) Get(key string) ([]byte, bool, error) {
	if len(key) != 64 {
		return nil, false, errors.New("invalid table key")
	}
	lo, hi := int64(0), t.count
	var record [76]byte
	for lo < hi {
		mid := (lo + hi) / 2
		if _, e := t.index.ReadAt(record[:], mid*76); e != nil {
			return nil, false, e
		}
		k := string(record[:64])
		if k < key {
			lo = mid + 1
		} else if k > key {
			hi = mid
		} else {
			n := binary.LittleEndian.Uint32(record[72:])
			if n > MaxRecord {
				return nil, false, errors.New("invalid table value length")
			}
			b := make([]byte, n)
			_, e := t.data.ReadAt(b, int64(binary.LittleEndian.Uint64(record[64:72])))
			return b, true, e
		}
	}
	return nil, false, nil
}
func (t *Table) Close() error { return errors.Join(t.index.Close(), t.data.Close()) }
