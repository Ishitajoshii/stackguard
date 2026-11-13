// Package scan streams explicitly selected roots without following symlinks.
package scan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Roots        []string
	Output, Temp string
	Workers      int
	Agent        string
	Exclude      []string
	Initial      []model.Identity
}
type result struct {
	ids   []model.Identity
	bytes int64
	err   bool
	skip  bool
}

// Each worker owns a 64 KiB buffer. The coordinator owns the spool and counters.
// The walker closes jobs, the worker waiter closes results; all sends select on
// ctx.Done. Results are bounded to twice the worker count.
func Run(ctx context.Context, o Options) (m model.Manifest, err error) {
	m = output.Manifest("scan")
	if o.Workers < 1 || o.Workers > 64 {
		return m, errors.New("workers must be 1..64")
	}
	if o.Agent == "" {
		o.Agent = "unbound"
	}
	if len(o.Roots) == 0 && len(o.Initial) == 0 {
		return m, errors.New("select at least one root or config")
	}
	if err = os.MkdirAll(o.Output, 0700); err != nil {
		return m, err
	}
	w, err := output.New(filepath.Join(o.Output, "inventory.jsonl"))
	if err != nil {
		return m, err
	}
	defer func() {
		err = errors.Join(err, w.Close())
		m.Ended = time.Now().UTC()
		m.Records = w.Records
		m.Complete = err == nil && m.Errors == 0
		err = errors.Join(err, output.Publish(filepath.Join(o.Output, "manifest.json"), m))
	}()
	dir, err := os.MkdirTemp(o.Temp, "stackguard-scan-*")
	if err != nil {
		return m, err
	}
	defer os.RemoveAll(dir)
	spool, err := os.Create(filepath.Join(dir, "spool"))
	if err != nil {
		return m, err
	}
	defer spool.Close()
	emit := func(id model.Identity) error {
		b, e := json.Marshal(id)
		if e != nil {
			return e
		}
		_, e = fmt.Fprintf(spool, "%s\t%s\n", id.ID, b)
		return e
	}
	for _, id := range o.Initial {
		if err = emit(id); err != nil {
			return m, err
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan string, o.Workers*2)
	results := make(chan result, o.Workers*2)
	var wg sync.WaitGroup
	send := func(r result) bool {
		select {
		case results <- r:
			return true
		case <-ctx.Done():
			return false
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		for _, root := range o.Roots {
			_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if e != nil {
					send(result{err: true})
					return nil
				}
				for _, x := range o.Exclude {
					if strings.EqualFold(d.Name(), x) {
						send(result{skip: true})
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
				if d.Type()&os.ModeSymlink != 0 {
					send(result{skip: true})
					return nil
				}
				if d.IsDir() {
					return nil
				}
				if !d.Type().IsRegular() {
					send(result{skip: true})
					return nil
				}
				select {
				case jobs <- p:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		}
	}()
	for i := 0; i < o.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case p, ok := <-jobs:
					if !ok {
						return
					}
					n, e := File(ctx, p, func(offset int64, match secrets.Match) error {
						v := model.Identity{Version: 1, ID: match.ID, Provider: match.Provider, Kind: "secret", Masked: match.Masked, Bindings: []model.Binding{{Agent: o.Agent, Location: secrets.Redact(filepath.Clean(p)) + fmt.Sprintf(":byte:%d", offset)}}}
						if !send(result{ids: []model.Identity{v}}) {
							return ctx.Err()
						}
						return nil
					})
					if !send(result{bytes: n, err: e != nil && !errors.Is(e, context.Canceled)}) {
						return
					}
				}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	for r := range results {
		m.Bytes += r.bytes
		if r.err {
			m.Errors++
		}
		if r.skip {
			m.Skipped++
		}
		for _, id := range r.ids {
			if e := emit(id); e != nil {
				err = e
				cancel()
			}
		}
	}
	if err != nil {
		return m, err
	}
	if err = ctx.Err(); err != nil {
		return m, err
	}
	if err = spool.Close(); err != nil {
		return m, err
	}
	sorted, err := pipeline.Sort(ctx, spool.Name(), dir)
	if err != nil {
		return m, err
	}
	f, err := os.Open(sorted)
	if err != nil {
		return m, err
	}
	defer f.Close()
	var current model.Identity
	var previous string
	flush := func() error {
		if current.ID == "" {
			return nil
		}
		sort.Slice(current.Bindings, func(i, j int) bool {
			a, _ := json.Marshal(current.Bindings[i])
			b, _ := json.Marshal(current.Bindings[j])
			return string(a) < string(b)
		})
		return w.Write(current)
	}
	err = pipeline.Lines(ctx, f, func(s string) error {
		if s == previous {
			return nil
		}
		previous = s
		_, b, ok := strings.Cut(s, "\t")
		if !ok {
			return errors.New("invalid spool")
		}
		var v model.Identity
		if e := json.Unmarshal([]byte(b), &v); e != nil {
			return e
		}
		if current.ID != v.ID {
			if e := flush(); e != nil {
				return e
			}
			current = v
			return nil
		}
		if len(current.Bindings) >= 4096 {
			return errors.New("identity exceeds 4096 bindings; narrow scan roots to preserve complete inventory")
		}
		current.Bindings = append(current.Bindings, v.Bindings...)
		return nil
	})
	if err == nil {
		err = flush()
	}
	m.Notes = []string{"No credential verification performed. Symlinks and special files skipped. Binary files scanned as bytes; UTF-16 secrets are unsupported. Identity binding limit: 4096; exceeding it makes this run incomplete."}
	return m, err
}

// File includes 4096 bytes of overlap, enough for all bounded signatures.
func File(ctx context.Context, path string, emit func(int64, secrets.Match) error) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.New("read input failed")
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	carry := ""
	var total, base int64
	for {
		if err = ctx.Err(); err != nil {
			return total, err
		}
		n, e := f.Read(buf)
		total += int64(n)
		text := carry + string(buf[:n])
		cut := len(text) - 4096
		if e == io.EOF {
			cut = len(text)
		}
		if cut < 0 {
			cut = 0
		}
		for _, m := range secrets.Detect(text, "") {
			if m.Start < cut {
				if err = emit(base+int64(m.Start), m); err != nil {
					return total, err
				}
			}
		}
		base += int64(cut)
		carry = text[cut:]
		if e == io.EOF {
			return total, nil
		}
		if e != nil {
			return total, errors.New("read input failed")
		}
	}
}
