// Package adapters keeps provider envelopes outside the graph. Native sources
// require explicit field mapping; absent identifiers are never inferred by time.
package adapters

import (
	"encoding/json"
	"errors"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"strconv"
	"time"
)

type TraceAdapter interface {
	Decode([]byte) ([]model.Event, error)
}
type Registry map[string]TraceAdapter

func DefaultRegistry() Registry {
	return Registry{"proxy": Normalized{}, "normalized": Normalized{}, "bridge": Normalized{}, "otlp-json": OTLP{}, "codex-otlp": Codex{}, "cloud-otlp": OTLPSpans{}, "cloudwatch-export": CloudWatch{}}
}

type Normalized struct{}

func (Normalized) Decode(b []byte) ([]model.Event, error) {
	var e model.Event
	if json.Unmarshal(b, &e) != nil {
		return nil, errors.New("invalid JSON event")
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return []model.Event{e}, nil
}

// OTLP reads exported log envelopes containing explicit stackguard.* attributes.
// This is a configurable collector contract, not a claim that native Codex emits
// those attributes. Real native schemas must be mapped after capture inspection.
type OTLP struct{}

func (OTLP) Decode(b []byte) ([]model.Event, error) {
	var v struct {
		ResourceLogs []struct {
			ScopeLogs []struct {
				LogRecords []struct {
					Time  string `json:"timeUnixNano"`
					Attrs []struct {
						Key   string `json:"key"`
						Value struct {
							String string `json:"stringValue"`
						} `json:"value"`
					} `json:"attributes"`
				} `json:"logRecords"`
			} `json:"scopeLogs"`
		} `json:"resourceLogs"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil, errors.New("invalid OTLP JSON")
	}
	var out []model.Event
	for _, r := range v.ResourceLogs {
		for _, s := range r.ScopeLogs {
			for _, l := range s.LogRecords {
				a := map[string]string{}
				for _, x := range l.Attrs {
					a[x.Key] = x.Value.String
				}
				n, err := strconv.ParseInt(l.Time, 10, 64)
				if err != nil {
					return nil, errors.New("invalid OTLP timestamp")
				}
				e := model.Event{Version: 1, TS: time.Unix(0, n).UTC(), Source: a["stackguard.source"], ID: a["stackguard.event_id"], Session: a["stackguard.session"], Span: a["stackguard.span"], Parent: a["stackguard.parent_span"], ParentSession: a["stackguard.parent_session"], Agent: a["stackguard.agent"], Kind: a["stackguard.kind"], Target: a["stackguard.target"], NHI: a["stackguard.nhi"], Attribution: a["stackguard.attribution"]}
				if err = e.Validate(); err != nil {
					return nil, errors.New("OTLP mapping incomplete; supply explicit source identifiers")
				}
				out = append(out, e)
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no mapped OTLP log records")
	}
	return out, nil
}

type CloudWatch struct{}

func (CloudWatch) Decode(b []byte) ([]model.Event, error) {
	var v struct {
		Events []struct {
			Message string `json:"message"`
			ID      string `json:"eventId"`
		} `json:"events"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil, errors.New("invalid CloudWatch export")
	}
	var out []model.Event
	for _, r := range v.Events {
		events, e := (Normalized{}).Decode([]byte(r.Message))
		if e != nil {
			events, e = (OTLPSpans{}).Decode([]byte(r.Message))
		}
		if e != nil {
			events, e = (AgentCoreSpan{}).Decode([]byte(r.Message))
		}
		if e != nil {
			return nil, errors.New("CloudWatch message needs a native schema adapter or normalized event")
		}
		for _, e := range events {
			e.Evidence = append(e.Evidence, "cloudwatch:"+r.ID)
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("empty CloudWatch export")
	}
	return out, nil
}
