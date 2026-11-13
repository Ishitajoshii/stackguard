package proxy

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type StdioOptions struct {
	Command   string
	Args, Env []string
	Input     io.ReadCloser
	Output    io.Writer
	Observer  *Observer
	Session   string
}

func Stdio(ctx context.Context, o StdioOptions) error {
	if o.Command == "" || o.Observer == nil {
		return errors.New("command and observer required")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, o.Command, o.Args...)
	cmd.Env = append(append([]string{}, o.Env...), "STACKGUARD_PROXY_SESSION="+o.Session)
	cmd.WaitDelay = 500 * time.Millisecond
	in, e := cmd.StdinPipe()
	if e != nil {
		return e
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		return e
	}
	cmd.Stderr = io.Discard // arbitrary child diagnostics may contain secrets
	if e = cmd.Start(); e != nil {
		return errors.New("start upstream failed")
	}
	inputDone := make(chan error, 1)
	outputDone := make(chan error, 1)
	uses := EnvironmentUses(o.Env)
	go func() {
		inputDone <- frames(ctx, o.Input, func(b []byte) error {
			if e := o.Observer.Observe(ctx, o.Session, b, false, uses); e != nil {
				return e
			}
			_, e := in.Write(b)
			return e
		})
		in.Close()
	}()
	go func() {
		outputDone <- frames(ctx, out, func(b []byte) error {
			if e := o.Observer.Observe(ctx, o.Session, b, true, nil); e != nil {
				return e
			}
			_, e := o.Output.Write(b)
			return e
		})
	}()
	var first error
	inputRead, outputRead := false, false
	select {
	case <-ctx.Done():
		first = ctx.Err()
	case first = <-outputDone:
		outputRead = true
	case first = <-inputDone:
		inputRead = true
		if first == nil {
			select {
			case first = <-outputDone:
				outputRead = true
			case <-ctx.Done():
				first = ctx.Err()
			}
		}
	}
	// Closing the upstream stdout reader or child exit must also release an idle
	// downstream input reader; otherwise an exited child leaves a goroutine behind.
	o.Input.Close()
	in.Close()
	out.Close()
	if first != nil {
		cancel()
		if output, ok := o.Output.(io.Closer); ok {
			output.Close()
		}
	}
	if !inputRead {
		<-inputDone
	}
	if !outputRead {
		<-outputDone
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-time.After(500 * time.Millisecond):
		cancel()
		waitErr = <-waitDone
		if first == nil {
			first = errors.New("upstream kept running after closing protocol output")
		}
	}
	cancel()
	if first != nil {
		return first
	}
	if waitErr != nil {
		return errors.New("upstream exited unsuccessfully")
	}
	return nil
}
func frames(ctx context.Context, r io.Reader, fn func([]byte) error) error {
	reader := bufio.NewReaderSize(r, 64<<10)
	var line []byte
	for {
		if e := ctx.Err(); e != nil {
			return e
		}
		b, e := reader.ReadSlice('\n')
		if len(line)+len(b) > 1<<20 {
			return errors.New("MCP frame exceeds 1 MiB")
		}
		line = append(line, b...)
		if e == bufio.ErrBufferFull {
			continue
		}
		if len(line) > 0 {
			if err := fn(line); err != nil {
				return err
			}
			line = nil
		}
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return errors.New("MCP pipe read failed")
		}
	}
}
