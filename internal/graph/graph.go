// Package graph resolves only explicit parent IDs using a disk-backed index.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"strings"
)

type Node struct {
	Agent, Parent, ParentSession string
	Ambiguous                    bool
}

func Key(session, span string) string { return secrets.Fingerprint(session + "\x00" + span) }
func NodeRecord(e model.Event) string {
	n := Node{Agent: e.Agent, Parent: e.Parent, ParentSession: e.ParentSession}
	if n.Parent != "" && n.ParentSession == "" {
		n.ParentSession = e.Session
	}
	b, _ := json.Marshal(n)
	return Key(e.Session, e.Span) + "\t" + string(b) + "\n"
}
func Index(ctx context.Context, path, dir string) (*pipeline.Table, error) {
	p, err := pipeline.Sort(ctx, path, dir)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out, err := os.CreateTemp(dir, "nodes-*")
	if err != nil {
		return nil, err
	}
	defer out.Close()
	var key string
	var node Node
	flush := func() error {
		if key == "" {
			return nil
		}
		b, _ := json.Marshal(node)
		_, e := fmt.Fprintf(out, "%s\t%s\n", key, b)
		return e
	}
	err = pipeline.Lines(ctx, f, func(line string) error {
		k, b, ok := strings.Cut(line, "\t")
		if !ok {
			return errors.New("invalid node spool")
		}
		var n Node
		if e := json.Unmarshal([]byte(b), &n); e != nil {
			return e
		}
		if k != key {
			if e := flush(); e != nil {
				return e
			}
			key = k
			node = n
			return nil
		}
		if node.Agent != n.Agent || node.Parent != "" && n.Parent != "" && (node.Parent != n.Parent || node.ParentSession != n.ParentSession) {
			node.Ambiguous = true
		}
		if node.Parent == "" {
			node.Parent = n.Parent
			node.ParentSession = n.ParentSession
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return nil, err
	}
	if err = out.Close(); err != nil {
		return nil, err
	}
	return pipeline.BuildTable(ctx, out.Name(), dir)
}
func Path(ctx context.Context, table *pipeline.Table, e model.Event) ([]string, string, error) {
	session, span := e.Session, e.Span
	seen := map[string]bool{}
	var path []string
	reason := ""
	for depth := 0; depth < 128; depth++ {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		key := Key(session, span)
		if seen[key] {
			reason = "parent cycle"
			break
		}
		seen[key] = true
		b, ok, err := table.Get(key)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			reason = "missing parent"
			break
		}
		var n Node
		if json.Unmarshal(b, &n) != nil {
			return nil, "", errors.New("invalid graph index")
		}
		if n.Ambiguous {
			reason = "ambiguous parent or agent mapping"
			break
		}
		path = append(path, n.Agent)
		if n.Parent == "" {
			break
		}
		span = n.Parent
		session = n.ParentSession
		if depth == 127 {
			reason = "lineage depth exceeds 128"
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if len(path) == 0 {
		path = []string{e.Agent}
	}
	return path, reason, nil
}
