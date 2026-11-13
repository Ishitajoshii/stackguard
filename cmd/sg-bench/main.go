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
	f.Parse(os.Args[1:])
	if *out == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
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
