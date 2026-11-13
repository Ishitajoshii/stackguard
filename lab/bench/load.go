package bench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/lab/mcp"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type LoadOptions struct {
	URL, Path, Output     string
	Concurrency, Requests int
}

func Load(ctx context.Context, o LoadOptions) (map[string]any, error) {
	if o.Concurrency < 1 || o.Concurrency > 2000 || o.Requests < o.Concurrency || o.Requests > 1000000 {
		return nil, errors.New("concurrency must be 1..2000; requests between concurrency and one million")
	}
	writer, e := output.New(filepath.Join(o.Output, "latencies.jsonl"))
	if e != nil {
		return nil, e
	}
	defer writer.Close()
	client, e := mcp.NewClient(o.URL, "benchmark", nil)
	if e != nil {
		return nil, e
	}
	args, _ := json.Marshal(map[string]string{"path": o.Path})
	var active, peak, serverPeak, failed, sequence atomic.Int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	durations := make([]time.Duration, 0, o.Requests)
	gate := make(chan struct{})
	start := time.Now()
	var writeErr error
	for i := 0; i < o.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			for {
				index := sequence.Add(1)
				if index > int64(o.Requests) || ctx.Err() != nil {
					return
				}
				n := active.Add(1)
				updateMax(&peak, n)
				begin := time.Now()
				result, err := client.Call(ctx, "repository_file", args)
				elapsed := time.Since(begin)
				active.Add(-1)
				if err != nil {
					failed.Add(1)
				} else {
					if value, ok := result.(map[string]any); ok {
						if p, ok := value["server_peak_inflight"].(float64); ok {
							updateMax(&serverPeak, int64(p))
						}
					}
				}
				mu.Lock()
				durations = append(durations, elapsed)
				reason := ""
				if err != nil {
					reason = err.Error()
				}
				if e := writer.Write(map[string]any{"request": index, "elapsed_ns": elapsed.Nanoseconds(), "success": err == nil, "error": reason}); e != nil && writeErr == nil {
					writeErr = e
				}
				mu.Unlock()
			}
		}()
	}
	close(gate)
	wg.Wait()
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	quantile := func(p float64) int64 {
		if len(durations) == 0 {
			return 0
		}
		i := int(float64(len(durations)-1) * p)
		return durations[i].Nanoseconds()
	}
	report := map[string]any{"schema_version": 1, "operation": "real_file_mcp_load", "requests": len(durations), "requested_concurrency": o.Concurrency, "client_peak_inflight": peak.Load(), "server_peak_inflight": serverPeak.Load(), "failures": failed.Load(), "p50_ns": quantile(.5), "p99_ns": quantile(.99), "elapsed_seconds": time.Since(start).Seconds(), "complete": ctx.Err() == nil && failed.Load() == 0 && writeErr == nil, "method": "Nearest lower order statistic. Compare matching baseline/proxy runs; subtract quantiles, not paired per-request latency. Barrier wait is included identically in both runs."}
	if e = output.Publish(filepath.Join(o.Output, "load.json"), report); e != nil {
		return report, e
	}
	if writeErr != nil {
		return report, writeErr
	}
	if ctx.Err() != nil {
		return report, ctx.Err()
	}
	if failed.Load() > 0 {
		return report, fmt.Errorf("%d load requests failed", failed.Load())
	}
	return report, nil
}
func updateMax(v *atomic.Int64, n int64) {
	for {
		old := v.Load()
		if n <= old || v.CompareAndSwap(old, n) {
			return
		}
	}
}
