package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// WorkloadGate verifies simultaneous tool handlers before reading real files.
// The identical gate must be used for baseline and proxied measurements.
type WorkloadGate struct {
	Required     int
	active, peak atomic.Int64
	once         sync.Once
	gate         chan struct{}
	Call         Call
}

func NewWorkload(root string, required int) *WorkloadGate {
	return &WorkloadGate{Required: required, gate: make(chan struct{}), Call: FileCall(root)}
}
func (w *WorkloadGate) Run(ctx context.Context, name string, args json.RawMessage) (any, error) {
	n := w.active.Add(1)
	defer w.active.Add(-1)
	for {
		p := w.peak.Load()
		if n <= p || w.peak.CompareAndSwap(p, n) {
			break
		}
	}
	if n >= int64(w.Required) {
		w.once.Do(func() { close(w.gate) })
	}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("concurrency barrier not reached within 30 seconds")
	case <-w.gate:
	}
	result, e := w.Call(ctx, name, args)
	if e != nil {
		return nil, e
	}
	v, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid file workload result")
	}
	v["server_peak_inflight"] = w.peak.Load()
	return v, nil
}
