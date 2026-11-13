// Package pipeline provides bounded, cancellable disk operations. Sort files
// contain secret-free, single-line records; callers own and remove the workspace.
package pipeline

import (
	"bufio"
	"container/heap"
	"context"
	"errors"
	"io"
	"os"
	"sort"
)

const MaxRecord = 1 << 20
const ChunkBytes = 8 << 20
const FanIn = 8

func Lines(ctx context.Context, r io.Reader, fn func(string) error) error {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64<<10), MaxRecord)
	for s.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(s.Text()); err != nil {
			return err
		}
	}
	if s.Err() != nil {
		return errors.New("read failed or record exceeds 1 MiB")
	}
	return ctx.Err()
}

// Sort uses at most an 8 MiB chunk, eight 1 MiB readers and bounded merge state.
// The on-disk run list itself is streamed, so metadata does not grow with input.
func Sort(ctx context.Context, input, dir string) (string, error) {
	f, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer f.Close()
	list, err := os.CreateTemp(dir, "runs-*")
	if err != nil {
		return "", err
	}
	listName := list.Name()
	defer func() { list.Close(); os.Remove(listName) }()
	chunk := make([]string, 0, 4096)
	size := 0
	count := 0
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		sort.Strings(chunk)
		out, e := os.CreateTemp(dir, "sort-*")
		if e != nil {
			return e
		}
		w := bufio.NewWriterSize(out, 64<<10)
		for _, line := range chunk {
			if _, e = w.WriteString(line + "\n"); e != nil {
				out.Close()
				return e
			}
		}
		e = errors.Join(w.Flush(), out.Close())
		if e == nil {
			_, e = io.WriteString(list, out.Name()+"\n")
		}
		chunk = nil
		size = 0
		count++
		return e
	}
	err = Lines(ctx, f, func(s string) error {
		chunk = append(chunk, s)
		size += len(s) + 16
		if size >= ChunkBytes {
			return flush()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if err = flush(); err != nil {
		return "", err
	}
	if count == 0 {
		empty, e := os.CreateTemp(dir, "sort-*")
		if e != nil {
			return "", e
		}
		return empty.Name(), empty.Close()
	}
	for count > 1 {
		if _, err = list.Seek(0, 0); err != nil {
			return "", err
		}
		next, e := os.CreateTemp(dir, "runs-*")
		if e != nil {
			return "", e
		}
		group := []string{}
		newCount := 0
		mergeGroup := func() error {
			p, e := merge(ctx, group, dir)
			if e != nil {
				return e
			}
			for _, old := range group {
				if e = os.Remove(old); e != nil {
					return e
				}
			}
			group = nil
			newCount++
			_, e = io.WriteString(next, p+"\n")
			return e
		}
		err = Lines(ctx, list, func(p string) error {
			group = append(group, p)
			if len(group) == FanIn {
				return mergeGroup()
			}
			return nil
		})
		if err == nil && len(group) > 0 {
			err = mergeGroup()
		}
		list.Close()
		os.Remove(listName)
		list = next
		listName = next.Name()
		count = newCount
		if err != nil {
			return "", err
		}
	}
	if _, err = list.Seek(0, 0); err != nil {
		return "", err
	}
	s := bufio.NewScanner(list)
	if !s.Scan() {
		return "", errors.New("missing sorted run")
	}
	return s.Text(), s.Err()
}

type item struct {
	line   string
	reader int
}
type items []item

func (h items) Len() int { return len(h) }
func (h items) Less(i, j int) bool {
	if h[i].line == h[j].line {
		return h[i].reader < h[j].reader
	}
	return h[i].line < h[j].line
}
func (h items) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *items) Push(x any)   { *h = append(*h, x.(item)) }
func (h *items) Pop() any     { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }
func merge(ctx context.Context, paths []string, dir string) (string, error) {
	out, err := os.CreateTemp(dir, "sort-*")
	if err != nil {
		return "", err
	}
	defer out.Close()
	w := bufio.NewWriterSize(out, 64<<10)
	readers := make([]*bufio.Scanner, len(paths))
	h := &items{}
	for i, p := range paths {
		f, e := os.Open(p)
		if e != nil {
			return "", e
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64<<10), MaxRecord)
		readers[i] = s
		if s.Scan() {
			heap.Push(h, item{s.Text(), i})
		} else if s.Err() != nil {
			return "", s.Err()
		}
	}
	for h.Len() > 0 {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		v := heap.Pop(h).(item)
		if _, err = w.WriteString(v.line + "\n"); err != nil {
			return "", err
		}
		s := readers[v.reader]
		if s.Scan() {
			heap.Push(h, item{s.Text(), v.reader})
		} else if s.Err() != nil {
			return "", s.Err()
		}
	}
	if err = w.Flush(); err != nil {
		return "", err
	}
	return out.Name(), nil
}
