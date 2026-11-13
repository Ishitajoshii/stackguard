package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/pipeline"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Risks treats absence as a coverage gap, never as proof of unauthorized use.
func Risks(ctx context.Context, dir, temp, inventory string) (err error) {
	work, e := os.MkdirTemp(temp, "stackguard-risks-*")
	if e != nil {
		return e
	}
	defer func() {
		if ctx.Err() == nil {
			err = errors.Join(err, os.RemoveAll(work))
		}
	}()
	bindings, e := os.Create(filepath.Join(work, "bindings"))
	if e != nil {
		return e
	}
	defer bindings.Close()
	read := func(path string, fn func(string) error) error {
		f, e := os.Open(path)
		if e != nil {
			return errors.New("open risk input failed")
		}
		defer f.Close()
		return pipeline.Lines(ctx, f, fn)
	}
	if inventory != "" {
		e = read(inventory, func(s string) error {
			var id model.Identity
			if json.Unmarshal([]byte(s), &id) != nil {
				return errors.New("invalid inventory")
			}
			if id.Unresolved {
				return nil
			}
			for _, b := range id.Bindings {
				if _, e := fmt.Fprintf(bindings, "%s\t1\n", secrets.Fingerprint(id.ID+"\x00"+b.Agent)); e != nil {
					return e
				}
			}
			return nil
		})
		if e != nil {
			return e
		}
	}
	if e = bindings.Close(); e != nil {
		return e
	}
	table, e := pipeline.BuildTable(ctx, bindings.Name(), work)
	if e != nil {
		return e
	}
	defer table.Close()
	used, e := os.Create(filepath.Join(work, "used"))
	if e != nil {
		return e
	}
	defer used.Close()
	spool, e := os.Create(filepath.Join(work, "findings"))
	if e != nil {
		return e
	}
	defer spool.Close()
	emit := func(rule, severity, id string, agents []string, reason, remedy string, refs []string) error {
		agents = union(nil, agents, 64)
		f := model.Finding{ID: secrets.Fingerprint(rule + "\x00" + id + "\x00" + strings.Join(agents, "\x00")), Rule: rule, Severity: severity, NHI: id, Agents: agents, Reason: reason, Remedy: remedy, Evidence: union(nil, refs, 64)}
		b, _ := json.Marshal(f)
		_, e := fmt.Fprintf(spool, "%s\t%s\n", f.ID, b)
		return e
	}
	var current string
	var agents, refs []string
	flush := func() error {
		if len(agents) < 3 {
			return nil
		}
		return emit("shared_identity", "medium", current, agents, "The same identity appears in calls attributed to at least three agents. Agent labels and capture coverage require review.", "Use separate least-privilege credentials for independently trusted agents.", refs)
	}
	e = read(filepath.Join(dir, "edges.jsonl"), func(s string) error {
		var edge model.Edge
		if json.Unmarshal([]byte(s), &edge) != nil {
			return errors.New("invalid edge")
		}
		if edge.NHI != current {
			if e := flush(); e != nil {
				return e
			}
			current = edge.NHI
			agents = nil
			refs = nil
		}
		agents = union(agents, []string{edge.Agent}, 64)
		refs = union(refs, edge.Evidence, 64)
		if _, e := fmt.Fprintf(used, "%s\t1\n", secrets.Fingerprint(edge.NHI)); e != nil {
			return e
		}
		if inventory != "" {
			_, ok, e := table.Get(secrets.Fingerprint(edge.NHI + "\x00" + edge.Agent))
			if e != nil {
				return e
			}
			if !ok {
				return emit("unwired_observation", "medium", edge.NHI, []string{edge.Agent}, "Observed use has no matching agent binding in the supplied inventory snapshot. This is missing wiring evidence, not proof of unauthorized use.", "Check snapshot timing, runtime injection and the intended binding.", edge.Evidence)
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	if e = flush(); e != nil {
		return e
	}
	if e = used.Close(); e != nil {
		return e
	}
	useTable, e := pipeline.BuildTable(ctx, used.Name(), work)
	if e != nil {
		return e
	}
	defer useTable.Close()
	if inventory != "" {
		e = read(inventory, func(s string) error {
			var id model.Identity
			if json.Unmarshal([]byte(s), &id) != nil {
				return errors.New("invalid inventory")
			}
			if id.Unresolved || (id.Kind != "secret" && id.Kind != "credential_set") {
				return nil
			}
			_, ok, e := useTable.Get(secrets.Fingerprint(id.ID))
			if e != nil || ok {
				return e
			}
			return emit("unused_in_window", "low", id.ID, nil, "No use was observed in the supplied capture. Incomplete telemetry or a different observation window can explain absence.", "Confirm coverage and expected usage before rotating or removing the credential.", nil)
		})
		if e != nil {
			return e
		}
	}
	var windowStart, windowEnd time.Time
	e = read(filepath.Join(dir, "events.jsonl"), func(s string) error {
		var event model.Event
		if json.Unmarshal([]byte(s), &event) != nil {
			return errors.New("invalid event")
		}
		if windowStart.IsZero() || event.TS.Before(windowStart) {
			windowStart = event.TS
		}
		if event.TS.After(windowEnd) {
			windowEnd = event.TS
		}
		if event.DelegatedFrom == "" || event.DelegatedFrom == event.Agent || len(event.Evidence) == 0 {
			return nil
		}
		uses := event.Credentials
		if event.NHI != "" {
			uses = append(uses, model.CredentialUse{ID: event.NHI, Attribution: event.Attribution})
		}
		for _, u := range uses {
			if u.Attribution != "direct" {
				continue
			}
			if e := emit("token_passthrough", "high", u.ID, []string{event.DelegatedFrom, event.Agent}, "An explicit credential-transfer observation names both agents and direct credential presence. Delegation alone does not trigger this rule.", "Keep credentials at the tool boundary; delegate scoped operations instead of passing tokens.", event.Evidence); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	if e = spool.Close(); e != nil {
		return e
	}
	sorted, e := pipeline.Sort(ctx, spool.Name(), work)
	if e != nil {
		return e
	}
	w, e := output.New(filepath.Join(dir, "findings.jsonl"))
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, w.Close()) }()
	last := ""
	return read(sorted, func(s string) error {
		k, b, _ := strings.Cut(s, "\t")
		if k == last {
			return nil
		}
		last = k
		var f model.Finding
		if json.Unmarshal([]byte(b), &f) != nil {
			return errors.New("invalid finding")
		}
		if !windowStart.IsZero() {
			f.WindowStart = &windowStart
			f.WindowEnd = &windowEnd
		}
		f.Coverage = "Supplied normalized sources only; event-time bounds are not proof of continuous or complete telemetry."
		return w.Write(f)
	})
}
