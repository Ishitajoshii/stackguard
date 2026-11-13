package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Ishitajoshii/stackguard/lab/bench"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	f := flag.NewFlagSet("sg-bench", flag.ExitOnError)
	endpoint := f.String("url", "http://127.0.0.1:8765/mcp", "baseline or proxied server")
	path := f.String("path", "README.md", "real repository file")
	out := f.String("out", "", "new output directory")
	concurrency := f.Int("concurrency", 1000, "concurrent requests")
	requests := f.Int("requests", 2000, "total calls")
	replay := f.String("replay", "", "normalized original event file; enables throughput replay mode")
	sourceManifest := f.String("source-manifest", "", "original capture manifest for replay provenance")
	events := f.Int64("events", 10000000, "throughput replay event count")
	sessions := f.String("sessions", "copies", "replay session shape: copies or single (throughput only)")
	f.Parse(os.Args[1:])
	if *out == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
	}
	if *replay != "" {
		e := bench.ReplayWithSessions(ctx, *replay, *sourceManifest, *out, *events, *sessions)
		if ctx.Err() != nil {
			os.Exit(17)
		}
		if e != nil {
			fmt.Fprintln(os.Stderr, "replay failed; check original normalized capture, manifest and new destination")
			os.Exit(2)
		}
		return
	}
	report, e := bench.Load(ctx, bench.LoadOptions{URL: *endpoint, Path: *path, Output: *out, Concurrency: *concurrency, Requests: *requests})
	if report != nil {
		json.NewEncoder(os.Stdout).Encode(report)
	}
	if ctx.Err() != nil {
		os.Exit(17)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, "benchmark incomplete; inspect result counts")
		os.Exit(2)
	}
}
