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
	cmd.Env = o.Env
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
	done := make(chan error, 2)
	uses := EnvironmentUses(o.Env)
	go func() {
		done <- frames(ctx, o.Input, func(b []byte) error {
			if e := o.Observer.Observe(ctx, o.Session, b, false, uses); e != nil {
				return e
			}
			_, e := in.Write(b)
			return e
		})
		in.Close()
	}()
	go func() {
		done <- frames(ctx, out, func(b []byte) error {
			if e := o.Observer.Observe(ctx, o.Session, b, true, nil); e != nil {
				return e
			}
			_, e := o.Output.Write(b)
			return e
		})
	}()
	var first error
	select {
	case <-ctx.Done():
		first = ctx.Err()
		cancel()
	case first = <-done:
		if first != nil {
			cancel()
		} else {
			select {
			case first = <-done:
			case <-ctx.Done():
				first = ctx.Err()
			}
			cancel()
			o.Input.Close()
			in.Close()
			out.Close()
			waitErr := cmd.Wait()
			if first != nil {
				return first
			}
			if waitErr != nil {
				return errors.New("upstream exited unsuccessfully")
			}
			return nil
		}
	}
	cancel()
	o.Input.Close()
	in.Close()
	out.Close()
	<-done
	_ = cmd.Wait()
	return first
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
