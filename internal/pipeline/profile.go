package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

const cpuProfileWindow = 30 * time.Second

// profileSink retains write errors because StopCPUProfile has no error result.
// The profiler alone writes it; the owner inspects it only after StopCPUProfile.
type profileSink struct {
	file *os.File
	err  error
}

func (s *profileSink) Write(b []byte) (int, error) {
	n, err := s.file.Write(b)
	s.err = errors.Join(s.err, err)
	return n, err
}

type profileSegment struct {
	File    string  `json:"file"`
	Seconds float64 `json:"wall_seconds"`
	Index   int     `json:"index"`
}

type cpuProfileSession struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
	err  error // read only after done closes
}

func (s *cpuProfileSession) finish() error {
	s.once.Do(func() { close(s.stop) })
	<-s.done
	return s.err
}

// Profiles are opt-in and never start an HTTP debug endpoint. CPU profiling
// rotates every 30 seconds: pprof retains sample metadata until finalization,
// so an unbounded recording can defeat a bounded data pipeline. The first
// segment uses cpu; later segments append .part-NNNNNN. A JSONL index records
// only finalized segments. Gaps during rotation are not sampled.
func Profiles(ctx context.Context, cpu, heap string) (func() error, error) {
	return profiles(ctx, cpu, heap, cpuProfileWindow)
}

func profiles(ctx context.Context, cpu, heap string, window time.Duration) (func() error, error) {
	var stopCPU func() error
	if cpu != "" {
		session, err := startCPUProfiles(ctx, cpu, window)
		if err != nil {
			return nil, err
		}
		stopCPU = session.finish
	}
	var once sync.Once
	var result error
	return func() error {
		once.Do(func() {
			if stopCPU != nil {
				result = stopCPU()
			}
			if heap != "" && ctx.Err() == nil {
				f, err := os.OpenFile(heap, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					result = errors.Join(result, errors.New("heap profile destination unavailable"))
					return
				}
				runtime.GC()
				result = errors.Join(result, pprof.WriteHeapProfile(f), f.Close())
			}
		})
		return result
	}, nil
}

func startCPUProfiles(ctx context.Context, path string, window time.Duration) (*cpuProfileSession, error) {
	if window <= 0 {
		return nil, errors.New("invalid CPU profile window")
	}
	open := func(name string) (*profileSink, error) {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, errors.New("CPU profile destination unavailable")
		}
		sink := &profileSink{file: f}
		if err = pprof.StartCPUProfile(sink); err != nil {
			f.Close()
			return nil, errors.New("CPU profiling unavailable")
		}
		return sink, nil
	}
	index, err := os.OpenFile(path+".segments.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("CPU profile index destination unavailable")
	}
	sink, err := open(path)
	if err != nil {
		index.Close()
		return nil, err
	}
	session := &cpuProfileSession{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		var result error
		defer func() {
			session.err = errors.Join(result, index.Sync(), index.Close())
			close(session.done)
		}()
		for number := 0; ; number++ {
			began := time.Now()
			timer := time.NewTimer(window)
			finish := false
			select {
			case <-ctx.Done():
				finish = true
			case <-session.stop:
				finish = true
			case <-timer.C:
			}
			timer.Stop()
			pprof.StopCPUProfile()
			result = errors.Join(sink.err, sink.file.Close())
			if result != nil {
				return
			}
			if result = json.NewEncoder(index).Encode(profileSegment{File: filepath.Base(sink.file.Name()), Seconds: time.Since(began).Seconds(), Index: number}); result != nil {
				return
			}
			if finish || ctx.Err() != nil {
				return
			}
			select {
			case <-session.stop:
				return
			default:
			}
			sink, result = open(fmt.Sprintf("%s.part-%06d", path, number+1))
			if result != nil {
				return
			}
		}
	}()
	return session, nil
}
