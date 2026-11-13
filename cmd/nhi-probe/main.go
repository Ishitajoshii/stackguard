package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

const usage = `nhi-probe: agent credential-consumption tracer

Usage:
  nhi-probe --help
  nhi-probe --version

The scan, proxy and trace commands are under development.
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
	// Do not echo unknown arguments: they may contain credential values.
	fmt.Fprintln(stderr, "unsupported command or arguments; run nhi-probe --help")
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
