package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/Ishitajoshii/stackguard/internal/pipeline"
)

func printLineage(ctx context.Context, path string, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	closed := make(chan struct{})
	stopClose := context.AfterFunc(ctx, func() {
		defer close(closed)
		f.Close()
		if c, ok := out.(io.Closer); ok {
			c.Close()
		}
	})
	defer func() {
		if !stopClose() {
			<-closed
		}
	}()
	err = pipeline.Lines(ctx, f, func(line string) error {
		n, err := io.WriteString(out, line+"\n")
		if err == nil && n != len(line)+1 {
			return io.ErrShortWrite
		}
		return err
	})
	return errors.Join(err, ctx.Err())
}
