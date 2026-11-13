package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/adapters"
	"github.com/Ishitajoshii/stackguard/internal/graph"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Input struct{ Adapter, Path string }
type Options struct {
	Inputs       []Input
	Output, Temp string
	Workers      int
	Lineage      bool
	Inventory    string
}

func Run(ctx context.Context, o Options) (m model.Manifest, err error) {
	m = output.Manifest("trace")
	if len(o.Inputs) == 0 || len(o.Inputs) > 128 || o.Workers < 1 || o.Workers > 64 {
		return m, errors.New("supply 1..128 inputs and 1..64 workers")
	}
	if err = os.MkdirAll(o.Output, 0700); err != nil {
		return m, err
	}
	events, err := output.New(filepath.Join(o.Output, "events.jsonl"))
	if err != nil {
		return m, err
	}
	defer func() {
		err = errors.Join(err, events.Close())
		m.Ended = time.Now().UTC()
		m.Complete = err == nil && m.Errors == 0
		m.Records = events.Records
		err = errors.Join(err, output.Publish(filepath.Join(o.Output, "manifest.json"), m))
	}()
	quarantine, err := output.New(filepath.Join(o.Output, "quarantine.jsonl"))
	if err != nil {
		return m, err
	}
	defer quarantine.Close()
	dir, err := os.MkdirTemp(o.Temp, "stackguard-trace-*")
	if err != nil {
		return m, err
	}
	defer os.RemoveAll(dir)
	spool, err := os.Create(filepath.Join(dir, "events"))
	if err != nil {
		return m, err
	}
	defer spool.Close()
	registry := adapters.DefaultRegistry()
	for _, input := range o.Inputs {
		adapter, ok := registry[input.Adapter]
		if !ok {
			return m, errors.New("unknown trace adapter")
		}
		source, e := output.ChecksumContext(ctx, input.Path)
		if e != nil {
			return m, e
		}
		m.Sources = append(m.Sources, source)
		m.Bytes += source.Bytes
		f, e := os.Open(input.Path)
		if e != nil {
			return m, errors.New("open trace input failed")
		}
		line := 0
		e = pipeline.Lines(ctx, f, func(s string) error {
			line++
			if strings.TrimSpace(s) == "" {
				return nil
			}
			records, e := adapter.Decode([]byte(s))
			if e != nil {
				m.Errors++
				return quarantine.Write(map[string]any{"source": source.Name, "line": line, "reason": "invalid or unmapped event; inspect private source"})
			}
			for _, event := range records {
				event.Evidence = append(event.Evidence, fmt.Sprintf("sha256:%s:line:%d", source.SHA256, line))
				sort.Strings(event.Evidence)
				b, _ := json.Marshal(event)
				b = []byte(secrets.Redact(string(b)))
				key := secrets.Fingerprint(event.Source + "\x00" + event.ID)
				if _, e = fmt.Fprintf(spool, "%s\t%s\n", key, b); e != nil {
					return e
				}
			}
			return nil
		})
		f.Close()
		if e != nil {
			return m, e
		}
	}
	if err = spool.Close(); err != nil {
		return m, err
	}
	sorted, err := pipeline.Sort(ctx, spool.Name(), dir)
	if err != nil {
		return m, err
	}
	nodes, err := os.Create(filepath.Join(dir, "nodes"))
	if err != nil {
		return m, err
	}
	defer nodes.Close()
	f, err := os.Open(sorted)
	if err != nil {
		return m, err
	}
	var key string
	var current model.Event
	flush := func() error {
		if key == "" {
			return nil
		}
		if err := events.Write(current); err != nil {
			return err
		}
		_, err := nodes.WriteString(graph.NodeRecord(current))
		return err
	}
	err = pipeline.Lines(ctx, f, func(s string) error {
		k, b, _ := strings.Cut(s, "\t")
		var e model.Event
		if json.Unmarshal([]byte(b), &e) != nil {
			return errors.New("invalid normalized spool")
		}
		if k != key {
			if err := flush(); err != nil {
				return err
			}
			key = k
			current = e
			return nil
		}
		a, aa := current, e
		a.Evidence = nil
		aa.Evidence = nil
		ab, _ := json.Marshal(a)
		bb, _ := json.Marshal(aa)
		if string(ab) != string(bb) {
			return errors.New("conflicting duplicate source event ID")
		}
		current.Evidence = union(current.Evidence, e.Evidence, 64)
		return nil
	})
	f.Close()
	if err == nil {
		err = flush()
	}
	if err != nil {
		return m, err
	}
	if err = nodes.Close(); err != nil {
		return m, err
	}
	table, err := graph.Index(ctx, nodes.Name(), dir)
	if err != nil {
		return m, err
	}
	defer table.Close()
	edgeSpool, err := os.Create(filepath.Join(dir, "edges"))
	if err != nil {
		return m, err
	}
	defer edgeSpool.Close()
	err = pipeline.Transform(ctx, filepath.Join(o.Output, "events.jsonl"), o.Workers, func(s string) string { var e model.Event; json.Unmarshal([]byte(s), &e); return e.Session }, func(s string) ([]string, error) {
		var e model.Event
		if json.Unmarshal([]byte(s), &e) != nil {
			return nil, errors.New("invalid normalized event")
		}
		if e.Kind != "tool_call" {
			return nil, nil
		}
		path, reason, err := graph.Path(ctx, table, e)
		if err != nil {
			return nil, err
		}
		uses := e.Credentials
		if e.NHI != "" {
			uses = append(uses, model.CredentialUse{ID: e.NHI, Attribution: e.Attribution})
		}
		var records []string
		for _, use := range uses {
			edge := model.Edge{NHI: use.ID, Agent: e.Agent, Path: path, Count: 1, First: e.TS, Last: e.TS, Attribution: []string{use.Attribution}, Evidence: e.Evidence, Unresolved: reason}
			identity, _ := json.Marshal([]any{edge.Agent, edge.Path, edge.Unresolved})
			call := e.Source + ":" + e.ID
			if e.CallID != "" {
				call = e.Session + ":" + e.CallID
			}
			b, _ := json.Marshal(edge)
			records = append(records, fmt.Sprintf("%s:%s\t%s\t%s\n", edge.NHI, secrets.Fingerprint(string(identity)), secrets.Fingerprint(call), b))
		}
		return records, nil
	}, edgeSpool)
	if err != nil {
		return m, err
	}
	if err = edgeSpool.Close(); err != nil {
		return m, err
	}
	sorted, err = pipeline.Sort(ctx, edgeSpool.Name(), dir)
	if err != nil {
		return m, err
	}
	edges, err := output.New(filepath.Join(o.Output, "edges.jsonl"))
	if err != nil {
		return m, err
	}
	defer edges.Close()
	var tree *os.File
	if o.Lineage {
		tree, err = os.OpenFile(filepath.Join(o.Output, "lineage.txt"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return m, err
		}
		defer tree.Close()
	}
	f, err = os.Open(sorted)
	if err != nil {
		return m, err
	}
	defer f.Close()
	key = ""
	lastCall := ""
	var edge model.Edge
	flushEdge := func() error {
		if key == "" {
			return nil
		}
		if tree != nil {
			if e := writeLineage(tree, edge); e != nil {
				return e
			}
		}
		return edges.Write(edge)
	}
	err = pipeline.Lines(ctx, f, func(s string) error {
		k, rest, _ := strings.Cut(s, "\t")
		call, b, _ := strings.Cut(rest, "\t")
		var v model.Edge
		if json.Unmarshal([]byte(b), &v) != nil {
			return errors.New("invalid edge spool")
		}
		if k != key {
			if e := flushEdge(); e != nil {
				return e
			}
			key = k
			edge = v
			lastCall = call
			return nil
		}
		if call != lastCall {
			edge.Count++
			lastCall = call
		}
		if v.First.Before(edge.First) {
			edge.First = v.First
		}
		if v.Last.After(edge.Last) {
			edge.Last = v.Last
		}
		edge.Attribution = union(edge.Attribution, v.Attribution, 4)
		edge.Evidence = union(edge.Evidence, v.Evidence, 64)
		return nil
	})
	if err == nil {
		err = flushEdge()
	}
	if err != nil {
		return m, err
	}
	m.Notes = []string{"call_count counts distinct observed attempts, including failed calls; completion records do not increment it.", "Duplicate source IDs must agree. Cross-source calls merge only with an explicit shared session/call_id and equal path.", "Evidence references per aggregate are capped at 64; full references remain in events.jsonl.", "Native Codex/AgentCore schema mapping and correctness require real original captures; envelope support alone is not validation.", "Session-partitioned workers resolve lineage against an immutable disk index; external sort and reduction are serial."}
	// Publish completion only after risk reduction succeeds as well.
	if err = Risks(ctx, o.Output, o.Temp, o.Inventory); err != nil {
		return m, err
	}
	return m, nil
}
func union(a, b []string, limit int) []string {
	all := append(append([]string{}, a...), b...)
	sort.Strings(all)
	out := all[:0]
	for _, s := range all {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
