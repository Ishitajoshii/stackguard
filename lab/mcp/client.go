package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/proxy"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Correlation struct{ Session, Span string }
type correlationKey struct{}
type requestKey struct{}

func RequestID(ctx context.Context) string { s, _ := ctx.Value(requestKey{}).(string); return s }

func WithCorrelation(ctx context.Context, c Correlation) context.Context {
	return context.WithValue(ctx, correlationKey{}, c)
}

type Client struct {
	URL      string
	Session  string
	Audit    func(any) error
	sequence atomic.Uint64
	http     *http.Client
}

func NewClient(endpoint, session string, audit func(any) error) (*Client, error) {
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "http" || !proxy.Loopback(u.Hostname()) || u.User != nil {
		return nil, errors.New("MCP endpoint must be an HTTP loopback URL without credentials")
	}
	return &Client{URL: endpoint, Session: session, Audit: audit, http: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Call(ctx context.Context, name string, args json.RawMessage) (any, error) {
	id := fmt.Sprintf("%s-%d", c.Session, c.sequence.Add(1))
	b, e := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	if e != nil {
		return nil, errors.New("invalid MCP arguments")
	}
	request, e := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", Protocol)
	request.Header.Set("X-Stackguard-Session", c.Session)
	parent, _ := ctx.Value(correlationKey{}).(Correlation)
	if parent.Span != "" {
		request.Header.Set("X-Stackguard-Parent-Session", parent.Session)
		request.Header.Set("X-Stackguard-Parent-Span", parent.Span)
	}
	response, e := c.http.Do(request)
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("local MCP service unavailable; start server and proxy")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("local MCP returned HTTP %d", response.StatusCode)
	}
	body, e := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if e != nil || len(body) > 1<<20 {
		return nil, errors.New("MCP response exceeds limit")
	}
	var v struct {
		ID     string          `json:"id"`
		Error  json.RawMessage `json:"error"`
		Result struct {
			Error   bool `json:"isError"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &v) != nil || v.ID != id {
		return nil, errors.New("invalid MCP response or mismatched request ID")
	}
	if c.Audit != nil {
		if e = c.Audit(map[string]any{"source": "bridge-correlation", "cloud_session": parent.Session, "cloud_tool_use_id": parent.Span, "downstream_session": c.Session, "downstream_request_id": secrets.Fingerprint(`"` + id + `"`), "tool": name, "checked_at": time.Now().UTC()}); e != nil {
			return nil, e
		}
	}
	if len(v.Error) > 0 && string(v.Error) != "null" || v.Result.Error {
		return nil, errors.New("read-only MCP tool failed; check provider access and query budget")
	}
	for _, item := range v.Result.Content {
		if item.Type == "text" {
			var value any
			if json.Unmarshal([]byte(item.Text), &value) != nil {
				return nil, errors.New("MCP tool did not return structured JSON")
			}
			return value, nil
		}
	}
	return nil, errors.New("MCP response contains no result")
}
