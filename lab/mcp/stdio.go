package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// Stdio keeps reading cancellation/control messages while a tool runs. At most
// eight tool requests are active; excess calls receive an explicit busy error.
// Blocking inputs must implement io.Closer so cancellation can interrupt reads.
func (s Server) Stdio(parent context.Context, in io.Reader, out io.Writer) (resultErr error) {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup
	var mu, writeMu sync.Mutex
	active := map[string]context.CancelFunc{}
	var writeErr error
	encoder := json.NewEncoder(out)
	closeInput := func() {
		if c, ok := in.(io.Closer); ok {
			c.Close()
		}
		if c, ok := out.(io.Closer); ok {
			c.Close()
		}
	}
	closed := make(chan struct{})
	stopClose := context.AfterFunc(ctx, func() {
		defer close(closed)
		closeInput()
	})
	defer func() {
		cancel()
		if stopClose() {
			closeInput()
		} else {
			<-closed
		}
		wg.Wait()
		writeMu.Lock()
		resultErr = errors.Join(resultErr, writeErr)
		writeMu.Unlock()
		if parent.Err() != nil {
			resultErr = errors.Join(resultErr, parent.Err())
		}
	}()
	send := func(response *Response) {
		if response == nil {
			return
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		if writeErr == nil {
			writeErr = encoder.Encode(response)
			if writeErr != nil {
				cancel()
			}
		}
	}
	reader := bufio.NewScanner(in)
	reader.Buffer(make([]byte, 64<<10), 1<<20)
	for reader.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		raw := append([]byte(nil), reader.Bytes()...)
		var request Request
		if json.Unmarshal(raw, &request) != nil || request.JSONRPC != "2.0" {
			return errors.New("invalid JSON-RPC request")
		}
		if len(request.ID) == 0 && request.Method == "notifications/cancelled" {
			var params struct {
				ID json.RawMessage `json:"requestId"`
			}
			if json.Unmarshal(request.Params, &params) != nil {
				return errors.New("invalid cancellation parameters")
			}
			mu.Lock()
			stop := active[string(params.ID)]
			mu.Unlock()
			if stop != nil {
				stop()
			}
			continue
		}
		if request.Method != "tools/call" || len(request.ID) == 0 {
			response, err := s.Handle(ctx, raw)
			if err != nil {
				return err
			}
			send(response)
			continue
		}
		id := string(request.ID)
		mu.Lock()
		_, duplicate := active[id]
		if duplicate || len(active) >= 8 {
			mu.Unlock()
			message := "too many active tool requests"
			if duplicate {
				message = "duplicate in-flight request ID"
			}
			send(&Response{JSONRPC: "2.0", ID: request.ID, Error: map[string]any{"code": -32000, "message": message}})
			continue
		}
		callCtx, stop := context.WithCancel(ctx)
		active[id] = stop
		mu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stop()
			response, err := s.Handle(callCtx, raw)
			if err != nil {
				response = &Response{JSONRPC: "2.0", ID: request.ID, Error: map[string]any{"code": -32600, "message": "invalid tool request"}}
			}
			mu.Lock()
			delete(active, id)
			mu.Unlock()
			send(response)
		}()
	}
	if reader.Err() != nil {
		return errors.New("MCP input failed or frame exceeds 1 MiB")
	}
	// EOF after valid requests allows their responses to finish. An interrupt or
	// output failure cancels ctx and therefore the outstanding calls.
	wg.Wait()
	return ctx.Err()
}
