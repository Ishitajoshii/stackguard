package pipeline

import (
	"context"
	"errors"
	"hash/fnv"
	"io"
	"os"
	"sync"
)

// Transform routes each session key to one bounded worker queue. The producer
// closes queues; the waiter closes results. The caller alone owns the output.
func Transform(ctx context.Context, path string, workers int, key func(string) string, fn func(string) ([]string, error), out io.Writer) error {
	if workers < 1 || workers > 64 {
		return errors.New("invalid worker count")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	type result struct {
		lines []string
		err   error
	}
	results := make(chan result, workers)
	queues := make([]chan string, workers)
	var wg sync.WaitGroup
	send := func(r result) {
		select {
		case results <- r:
		case <-ctx.Done():
		}
	}
	for i := range queues {
		q := make(chan string, 2)
		queues[i] = q
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case s, ok := <-q:
					if !ok {
						return
					}
					lines, e := fn(s)
					send(result{lines, e})
					if e != nil {
						return
					}
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			for _, q := range queues {
				close(q)
			}
		}()
		e := Lines(ctx, f, func(s string) error {
			h := fnv.New32a()
			h.Write([]byte(key(s)))
			select {
			case queues[int(h.Sum32())%workers] <- s:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		if e != nil {
			send(result{err: e})
		}
	}()
	go func() { wg.Wait(); close(results) }()
	var first error
	for r := range results {
		if r.err != nil && first == nil {
			first = r.err
			cancel()
		}
		if first != nil {
			continue
		}
		for _, line := range r.lines {
			if _, e = io.WriteString(out, line); e != nil {
				first = e
				cancel()
				break
			}
		}
	}
	if first != nil {
		return first
	}
	return ctx.Err()
}
