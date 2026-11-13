// Package proxy observes MCP metadata while preserving protocol bodies.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"
)

type observation struct {
	session  string
	raw      []byte
	response bool
	uses     []model.CredentialUse
	ack      chan error
}
type pending struct{ span, call string }

// kestrelShard is owned exclusively by its observer worker. There is no global
// session lock. Pending calls are bounded and removed on completion.
type kestrelShard struct {
	pending  map[string]pending
	sequence uint64
}
type Observer struct {
	ctx                context.Context
	queues             []chan observation
	wg                 sync.WaitGroup
	run, agent, client string
	write              func(any) error
}

func NewObserver(ctx context.Context, workers int, run, agent, client string, write func(any) error) (*Observer, error) {
	if workers < 1 || workers > 64 {
		return nil, errors.New("workers must be 1..64")
	}
	o := &Observer{ctx: ctx, run: run, agent: agent, client: client, write: write}
	for i := 0; i < workers; i++ {
		q := make(chan observation, 8)
		o.queues = append(o.queues, q)
		o.wg.Add(1)
		go func(index int) {
			defer o.wg.Done()
			s := kestrelShard{pending: map[string]pending{}}
			for {
				select {
				case <-ctx.Done():
					return
				case v := <-q:
					v.ack <- o.process(&s, index, v)
				}
			}
		}(i)
	}
	return o, nil
}

// Wait is called only after cancelling the observer context.
func (o *Observer) Wait() { o.wg.Wait() }
func (o *Observer) Observe(ctx context.Context, session string, raw []byte, response bool, uses []model.CredentialUse) error {
	if len(raw) > 1<<20 {
		return errors.New("MCP frame exceeds 1 MiB")
	}
	h := fnv.New32a()
	h.Write([]byte(session))
	q := o.queues[int(h.Sum32())%len(o.queues)]
	v := observation{session: session, raw: append([]byte(nil), raw...), response: response, uses: uses, ack: make(chan error, 1)}
	select {
	case q <- v:
	case <-ctx.Done():
		return ctx.Err()
	case <-o.ctx.Done():
		return o.ctx.Err()
	}
	select {
	case e := <-v.ack:
		return e
	case <-ctx.Done():
		return ctx.Err()
	case <-o.ctx.Done():
		return o.ctx.Err()
	}
}
func (o *Observer) process(s *kestrelShard, index int, v observation) error {
	var m struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
		Error  json.RawMessage `json:"error"`
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if json.Unmarshal(v.raw, &m) != nil {
		return errors.New("invalid JSON-RPC observation")
	}
	if len(m.ID) == 0 || string(m.ID) == "null" {
		return nil
	}
	key := v.session + "\x00" + string(m.ID)
	event := model.Event{Version: 1, TS: time.Now().UTC(), Source: "proxy", Session: v.session, Agent: o.agent, Client: o.client, Evidence: []string{"proxy:" + o.run}}
	if !v.response {
		if m.Method != "tools/call" {
			return nil
		}
		if len(s.pending) >= 4096 {
			return errors.New("pending call limit exceeded")
		}
		if _, ok := s.pending[key]; ok {
			return errors.New("duplicate in-flight JSON-RPC ID")
		}
		s.sequence++
		span := secrets.Fingerprint(fmt.Sprintf("%s:%d:%d", o.run, index, s.sequence))
		s.pending[key] = pending{span: span, call: span}
		event.ID = span
		event.Span = span
		event.CallID = span
		event.Kind = "tool_call"
		event.Status = "attempted"
		event.Target = secrets.Redact(m.Params.Name)
		event.Credentials = append(event.Credentials, v.uses...)
		var args any
		if len(m.Params.Arguments) > 0 && json.Unmarshal(m.Params.Arguments, &args) == nil {
			findUses(args, "", &event.Credentials)
		}
		sort.Slice(event.Credentials, func(i, j int) bool { return event.Credentials[i].ID < event.Credentials[j].ID })
		unique := event.Credentials[:0]
		for _, c := range event.Credentials {
			if len(unique) == 0 || unique[len(unique)-1] != c {
				unique = append(unique, c)
			}
		}
		event.Credentials = unique
	} else {
		p, ok := s.pending[key]
		if !ok {
			return nil
		}
		delete(s.pending, key)
		event.ID = p.span + ":completion"
		event.Span = p.span
		event.CallID = p.call
		event.Kind = "tool_completion"
		event.Status = "succeeded"
		if len(m.Error) > 0 && string(m.Error) != "null" || m.Result.IsError {
			event.Status = "failed"
		}
	}
	if e := event.Validate(); e != nil {
		return e
	}
	return o.write(event)
}
func findUses(v any, key string, out *[]model.CredentialUse) {
	if len(*out) > 64 {
		return
	}
	switch x := v.(type) {
	case string:
		for _, m := range secrets.Detect(x, key) {
			*out = append(*out, model.CredentialUse{ID: m.ID, Attribution: "direct"})
		}
	case map[string]any:
		for k, value := range x {
			findUses(value, k, out)
		}
	case []any:
		for _, value := range x {
			findUses(value, key, out)
		}
	}
}
func EnvironmentUses(env []string) []model.CredentialUse {
	var out []model.CredentialUse
	awsValues := map[string]string{}
	for _, entry := range env {
		k, v, _ := strings.Cut(entry, "=")
		if secrets.AWSField(k) {
			awsValues[k] = v
		}
	}
	if id, ok := secrets.AWSSet(awsValues["AWS_ACCESS_KEY_ID"], awsValues["AWS_SECRET_ACCESS_KEY"], awsValues["AWS_SESSION_TOKEN"]); ok {
		out = append(out, model.CredentialUse{ID: id, Attribution: "implicit"})
	}
	for _, entry := range env {
		k, v, _ := strings.Cut(entry, "=")
		if secrets.AWSField(k) {
			continue
		}
		for _, m := range secrets.Detect(v, k) {
			out = append(out, model.CredentialUse{ID: m.ID, Attribution: "implicit"})
		}
	}
	return out
}
