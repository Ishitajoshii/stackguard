package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/proxy"
	"github.com/Ishitajoshii/stackguard/lab/telemetry"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:4318", "loopback collector")
	out := flag.String("out", "", "new capture directory")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	w, e := output.New(filepath.Join(*out, "native-otlp.jsonl"))
	if e != nil {
		fmt.Fprintln(os.Stderr, "capture destination unavailable")
		os.Exit(2)
	}
	m := output.Manifest("native_otlp_metadata_capture")
	m.Notes = []string{"Sender-exported identifiers and timestamps retained. Prompts, argument/output values, account metadata and bodies discarded at collection; exporter must use OTLP/HTTP JSON.", "Native exporter enablement and coverage must be verified separately."}
	e = proxy.Serve(ctx, *listen, telemetry.Handler(w.Write))
	e = errors.Join(e, w.Close())
	m.Ended = time.Now().UTC()
	m.Records = w.Records
	m.Complete = e == nil
	if e != nil {
		m.Errors = 1
	}
	publishErr := output.Publish(filepath.Join(*out, "manifest.json"), m)
	if publishErr != nil {
		os.Exit(2)
	}
	if ctx.Err() != nil {
		os.Exit(17)
	}
	if e != nil {
		os.Exit(2)
	}
}
