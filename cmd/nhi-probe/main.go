package main

import (
	"context"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/cli"
	"io"
	"os"
	"os/signal"
)

var version = "dev"

const usage = `nhi-probe: agent credential-consumption tracer

Usage:
  nhi-probe --help
  nhi-probe --version
  nhi-probe scan --root PATH --out DIR [--config codex=PATH] [--workers 4]
  nhi-probe proxy --mode stdio --out DIR [--agent unknown] -- COMMAND ARGS...
  nhi-probe proxy --mode http --upstream URL --listen 127.0.0.1:8765 --out DIR
  nhi-probe trace --input normalized=PATH --out DIR [--inventory PATH] [--lineage]

Outputs refuse overwrite. Use a new directory for each run. Ctrl-C exits 17.
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	return cli.Run(context.Background(), args, os.Stdin, stdout, stderr)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "--version" {
		os.Exit(run(args, os.Stdout, os.Stderr))
	}
	os.Exit(cli.Run(ctx, args, os.Stdin, os.Stdout, os.Stderr))
}
