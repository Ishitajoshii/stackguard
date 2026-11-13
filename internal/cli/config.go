package cli

import (
	"errors"
	"io"
	"os"
)

func boundedConfig(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if len(b) > 1<<20 {
		return nil, errors.New("configuration exceeds 1 MiB")
	}
	return b, err
}
