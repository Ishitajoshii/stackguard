// Package mcp implements the lab's read-only MCP tool server. It is stateless
// Streamable HTTP: POST JSON responses, notification 202, GET/DELETE 405.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/correlation"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"github.com/Ishitajoshii/stackguard/lab/scout"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const Protocol = "2025-06-18"

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"inputSchema"`
}

func Tools() []Tool {
	schema := func(properties map[string]any, required []string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "integer", "minimum": 1}
	return []Tool{{"github_search", "Search public repositories; bounded by configured budget", schema(map[string]any{"interest": str}, []string{"interest"})}, {"github_repo", "Read public repository metadata", schema(map[string]any{"repository": str}, []string{"repository"})}, {"github_issues", "Read candidate open unassigned issues", schema(map[string]any{"repository": str}, []string{"repository"})}, {"github_issue", "Read one issue and check active linked pull requests", schema(map[string]any{"repository": str, "number": num}, []string{"repository", "number"})}, {"notion_notes", "Read only the configured dedicated Notion page subtree", schema(map[string]any{}, nil)}}
}

type Call func(context.Context, string, json.RawMessage) (any, error)
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   any             `json:"error,omitempty"`
}
type Server struct {
	Call  Call
	Tools []Tool
}

func (s Server) Handle(ctx context.Context, b []byte) (*Response, error) {
	if len(b) > 1<<20 {
		return nil, errors.New("MCP request exceeds 1 MiB")
	}
	var r Request
	if json.Unmarshal(b, &r) != nil || r.JSONRPC != "2.0" {
		return nil, errors.New("invalid JSON-RPC request")
	}
	if len(r.ID) == 0 {
		return nil, nil
	}
	res := &Response{JSONRPC: "2.0", ID: r.ID}
	switch r.Method {
	case "initialize":
		res.Result = map[string]any{"protocolVersion": Protocol, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "stackguard-readonly-lab", "version": "1"}}
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = map[string]any{"tools": s.Tools}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if json.Unmarshal(r.Params, &p) != nil {
			res.Error = map[string]any{"code": -32602, "message": "invalid tool parameters"}
			break
		}
		allowed := false
		for _, tool := range s.Tools {
			if tool.Name == p.Name {
				allowed = true
			}
		}
		if !allowed {
			res.Error = map[string]any{"code": -32602, "message": "tool is not allowed"}
			break
		}
		boundary := correlation.From(ctx)
		boundary.RequestID = secrets.Fingerprint(string(r.ID))
		ctx = correlation.With(ctx, boundary)
		v, e := s.Call(ctx, p.Name, p.Arguments)
		if e != nil {
			res.Result = map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": secrets.Redact(e.Error())}}}
			break
		}
		encoded, e := json.Marshal(v)
		if e != nil || len(encoded) > 512<<10 {
			res.Result = map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": "tool response exceeds limit or cannot be encoded"}}}
			break
		}
		res.Result = map[string]any{"content": []any{map[string]any{"type": "text", "text": secrets.Redact(string(encoded))}}}
	default:
		res.Error = map[string]any{"code": -32601, "message": "method not found"}
	}
	return res, nil
}
func (s Server) HTTP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "origin rejected", 403)
			return
		}
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "stateless endpoint supports POST", 405)
			return
		}
		if version := r.Header.Get("Mcp-Protocol-Version"); version != "" && version != Protocol {
			http.Error(w, "unsupported protocol version", 400)
			return
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if e != nil {
			http.Error(w, "read failed", 400)
			return
		}
		res, e := s.Handle(correlation.With(r.Context(), correlation.Boundary{Session: r.Header.Get("X-Stackguard-Session")}), b)
		if e != nil {
			http.Error(w, "invalid MCP request", 400)
			return
		}
		if res == nil {
			w.WriteHeader(202)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	})
}
func (s Server) Stdio(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewScanner(in)
	reader.Buffer(make([]byte, 64<<10), 1<<20)
	encoder := json.NewEncoder(out)
	for reader.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res, e := s.Handle(ctx, reader.Bytes())
		if e != nil {
			return e
		}
		if res != nil {
			if e = encoder.Encode(res); e != nil {
				return e
			}
		}
	}
	return reader.Err()
}
func ProviderCall(gh, notion *scout.API, p scout.Profile) Call {
	return func(ctx context.Context, name string, args json.RawMessage) (any, error) {
		var a struct {
			Interest   string `json:"interest"`
			Repository string `json:"repository"`
			Number     int    `json:"number"`
		}
		if len(args) == 0 {
			args = []byte("{}")
		}
		decoder := json.NewDecoder(strings.NewReader(string(args)))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&a) != nil {
			return nil, errors.New("invalid tool arguments")
		}
		switch name {
		case "github_search":
			return gh.Search(ctx, a.Interest, p.Candidates)
		case "github_repo":
			return gh.Repo(ctx, a.Repository)
		case "github_issues":
			return gh.Issues(ctx, a.Repository, p.Issues)
		case "github_issue":
			issue, e := gh.Issue(ctx, a.Repository, a.Number)
			if e != nil {
				return nil, e
			}
			active, complete, e := gh.ActivePR(ctx, a.Repository, a.Number)
			if e != nil {
				return nil, e
			}
			return map[string]any{"issue": issue, "active_linked_pr": active, "linkage_scan_complete": complete}, nil
		case "notion_notes":
			if p.Notion == "" {
				return nil, errors.New("Notion page not configured")
			}
			return notion.Notes(ctx, p.Notion)
		default:
			return nil, errors.New("unknown read-only tool")
		}
	}
}

// FileCall supports real local repository workloads for transport measurement.
// It never executes repository code or follows a symlink outside the root.
func FileCall(root string) Call {
	return func(ctx context.Context, name string, args json.RawMessage) (any, error) {
		if name != "repository_file" {
			return nil, errors.New("unknown file tool")
		}
		var a struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(args, &a) != nil || a.Path == "" || filepath.IsAbs(a.Path) {
			return nil, errors.New("relative repository file required")
		}
		resolvedRoot, e := filepath.EvalSymlinks(root)
		if e != nil {
			return nil, errors.New("repository root unavailable")
		}
		target, e := filepath.EvalSymlinks(filepath.Join(resolvedRoot, a.Path))
		if e != nil {
			return nil, errors.New("repository file unavailable")
		}
		rel, e := filepath.Rel(resolvedRoot, target)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, errors.New("file is outside repository root")
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		f, e := os.Open(target)
		if e != nil {
			return nil, errors.New("open repository file failed")
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, 256<<10))
		if e != nil {
			return nil, errors.New("read repository file failed")
		}
		return map[string]any{"bytes_read": len(b), "sha256": secrets.Fingerprint(string(b)), "path": fmt.Sprint(filepath.ToSlash(rel)), "read_limit_bytes": 256 << 10}, nil
	}
}
