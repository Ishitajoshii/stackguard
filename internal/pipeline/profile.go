package pipeline

import (
	"context"
	"errors"
	"os"
	"runtime"
	"runtime/pprof"
)

// Profiles are opt-in and never start an HTTP debug endpoint.
func Profiles(ctx context.Context, cpu, heap string) (func() error, error) {
	var cpuFile *os.File
	if cpu != "" {
		f, e := os.OpenFile(cpu, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, errors.New("CPU profile destination unavailable")
		}
		if e = pprof.StartCPUProfile(f); e != nil {
			f.Close()
			return nil, errors.New("CPU profiling unavailable")
		}
		cpuFile = f
	}
	return func() error {
		var err error
		if cpuFile != nil {
			pprof.StopCPUProfile()
			err = cpuFile.Close()
		}
		if heap != "" && ctx.Err() == nil {
			f, e := os.OpenFile(heap, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return errors.Join(err, errors.New("heap profile destination unavailable"))
			}
			runtime.GC()
			err = errors.Join(err, pprof.WriteHeapProfile(f), f.Close())
		}
		return err
	}, nil
}
