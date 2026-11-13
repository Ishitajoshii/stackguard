package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/config"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/proxy"
	"github.com/Ishitajoshii/stackguard/internal/scan"
	"github.com/Ishitajoshii/stackguard/internal/trace"
	"github.com/Ishitajoshii/stackguard/internal/verify"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type values []string

func (v *values) String() string     { return strings.Join(*v, ",") }
func (v *values) Set(s string) error { *v = append(*v, s); return nil }
func Run(ctx context.Context, args []string, stdin io.ReadCloser, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printUsage(stdout)
		return 0
	}
	err := execute(ctx, args, stdin, stdout)
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "interrupted; inspect the partial manifest")
		return 17
	}
	fmt.Fprintln(stderr, "operation failed:", safeError(err))
	return 2
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, `Usage: nhi-probe <scan|proxy|trace> [options]

  scan   Find redacted credential candidates in explicit roots/configs
  proxy  Forward MCP over stdio or Streamable HTTP and record observations
  trace  Normalize captures, resolve credential paths and report risks

Run nhi-probe <command> --help for command options.
Choose a fresh --out directory. Ctrl-C writes partial output and exits 17.`)
}

// Do not echo OS/provider errors, paths or flag values, which can contain secrets.
func safeError(err error) string {
	switch {
	case errors.Is(err, os.ErrExist):
		return "destination already exists"
	default:
		return "check command options, input schema, permissions and output destination; no raw provider diagnostics are logged"
	}
}
func execute(ctx context.Context, args []string, stdin io.ReadCloser, stdout io.Writer) (resultErr error) {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.Usage = func() {
		// Called only by the known command branches below. Defaults are static;
		// parsing errors and supplied values still go to io.Discard.
		fmt.Fprintf(stdout, "Usage: nhi-probe %s [options]\n", args[0])
		f.SetOutput(stdout)
		f.PrintDefaults()
		f.SetOutput(io.Discard)
	}
	out := f.String("out", "", "new output directory")
	temp := f.String("temp", "", "temporary/spill directory (defaults to OS TEMP)")
	workers := f.Int("workers", 4, "bounded worker count")
	cpu := f.String("cpu-profile", "", "new CPU profile file")
	heap := f.String("heap-profile", "", "new heap profile file; skipped on interruption")
	var profileFinish func() error
	defer func() {
		if profileFinish != nil {
			resultErr = errors.Join(resultErr, profileFinish())
		}
	}()
	startProfiles := func() error {
		finish, e := pipeline.Profiles(ctx, *cpu, *heap)
		if e != nil {
			return e
		}
		profileFinish = finish
		return nil
	}
	switch args[0] {
	case "scan":
		verifyOwned := f.String("verify-owned", "", "opt-in owned fingerprint/environment allowlist JSON")
		var roots, configs, exclude values
		f.Var(&roots, "root", "input root; repeatable")
		f.Var(&configs, "config", "parser=path; repeatable")
		f.Var(&exclude, "exclude", "excluded basename; repeatable")
		agent := f.String("agent", "unbound", "agent binding for file findings")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if *out == "" || len(f.Args()) != 0 {
			return errors.New("output required")
		}
		var initial []model.Identity
		if e := startProfiles(); e != nil {
			return e
		}
		for _, entry := range configs {
			name, path, ok := strings.Cut(entry, "=")
			parser := config.DefaultRegistry()[name]
			if !ok || parser == nil {
				return errors.New("config needs codex=path or agentcore=path")
			}
			b, e := boundedConfig(path)
			if e != nil {
				return errors.New("read config failed")
			}
			d, e := parser.Parse(b)
			if e != nil {
				return e
			}
			initial = append(initial, config.Resolve(d, path, os.LookupEnv)...)
		}
		m, e := scan.Run(ctx, scan.Options{Roots: roots, Initial: initial, Output: *out, Temp: *temp, Workers: *workers, Agent: *agent, Exclude: append([]string{".git", ".work", "private", "profiles"}, exclude...)})
		if e == nil && !m.Complete {
			return errors.New("scan incomplete; inspect manifest")
		}
		if e == nil && *verifyOwned != "" {
			return verify.Run(ctx, filepath.Join(*out, "inventory.jsonl"), *verifyOwned, *out)
		}
		return e
	case "trace":
		var inputs values
		f.Var(&inputs, "input", "adapter=path; repeatable")
		inventory := f.String("inventory", "", "inventory JSONL snapshot")
		lineage := f.Bool("lineage", false, "print deterministic lineage trees and save lineage.txt")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if *out == "" || len(f.Args()) != 0 {
			return errors.New("output required")
		}
		var source []trace.Input
		if e := startProfiles(); e != nil {
			return e
		}
		for _, entry := range inputs {
			kind, path, ok := strings.Cut(entry, "=")
			if !ok {
				return errors.New("input needs adapter=path")
			}
			source = append(source, trace.Input{Adapter: kind, Path: path})
		}
		m, e := trace.Run(ctx, trace.Options{Inputs: source, Output: *out, Temp: *temp, Workers: *workers, Lineage: *lineage, Inventory: *inventory})
		if e != nil {
			return e
		}
		if !m.Complete {
			return errors.New("trace contains quarantined records")
		}
		if *lineage {
			return printLineage(ctx, filepath.Join(*out, "lineage.txt"), stdout)
		}
		return nil
	case "proxy":
		mode := f.String("mode", "stdio", "stdio or http")
		upstream := f.String("upstream", "", "fixed MCP upstream URL")
		listen := f.String("listen", "127.0.0.1:8765", "loopback listen address")
		agent := f.String("agent", "unknown", "configured initiating-agent label, not inferred identity")
		client := f.String("client", "unknown", "executing client label")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if *out == "" {
			return errors.New("output required")
		}
		w, e := output.New(filepath.Join(*out, "proxy.jsonl"))
		if profileErr := startProfiles(); profileErr != nil {
			if w != nil {
				w.Close()
			}
			return profileErr
		}
		if e != nil {
			return e
		}
		m := output.Manifest("proxy")
		observeCtx, cancel := context.WithCancel(ctx)
		observer, e := proxy.NewObserver(observeCtx, *workers, m.RunID, *agent, *client, w.Write)
		if e != nil {
			cancel()
			w.Close()
			return e
		}
		if *mode == "stdio" {
			command := f.Args()
			if len(command) == 0 {
				e = errors.New("upstream command required")
			} else {
				e = proxy.Stdio(ctx, proxy.StdioOptions{Command: command[0], Args: command[1:], Env: os.Environ(), Input: stdin, Output: stdout, Observer: observer, Session: m.RunID})
			}
		} else if *mode == "http" {
			handler, handlerErr := proxy.HTTP(*upstream, observer)
			if handlerErr != nil {
				e = handlerErr
			} else {
				e = proxy.Serve(ctx, *listen, handler)
			}
		} else {
			e = errors.New("unknown proxy mode")
		}
		cancel()
		observer.Wait()
		e = errors.Join(e, w.Close())
		m.Ended = time.Now().UTC()
		m.Records = w.Records
		m.Complete = e == nil
		if e != nil {
			m.Errors = 1
		}
		m.Notes = []string{"Agent/client labels are configured assertions. Credential presence in child environment is implicit use, not proof of provider API consumption.", "Single call events contain credential_uses. Completion records carry success/failure separately."}
		return errors.Join(e, output.Publish(filepath.Join(*out, "manifest.json"), m))
	default:
		return errors.New("unknown command")
	}
}
