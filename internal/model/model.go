// Package model contains versioned, secret-free interchange records.
package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const Version = 1

type Binding struct {
	Agent     string `json:"agent"`
	Server    string `json:"server"`
	Location  string `json:"location"`
	Reference string `json:"reference,omitempty"`
}

type Identity struct {
	Version    int       `json:"schema_version"`
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	Kind       string    `json:"identity_kind"`
	Masked     string    `json:"masked,omitempty"`
	Bindings   []Binding `json:"bindings"`
	Unresolved bool      `json:"unresolved,omitempty"`
}

type Event struct {
	Version         int       `json:"schema_version"`
	ID              string    `json:"event_id"`
	TS              time.Time `json:"ts"`
	Source          string    `json:"source"`
	Session         string    `json:"session"`
	Span            string    `json:"span"`
	Parent          string    `json:"parent_span,omitempty"`
	ParentSession   string    `json:"parent_session,omitempty"`
	RequestID       string    `json:"request_id,omitempty"`
	ParentRequestID string    `json:"parent_request_id,omitempty"`
	Agent           string    `json:"agent"`
	Kind            string    `json:"kind"`
	Target          string    `json:"target,omitempty"`
	NHI             string    `json:"nhi,omitempty"`
	IdentityKind    string    `json:"identity_kind,omitempty"`
	Attribution     string    `json:"attribution_kind,omitempty"`
	Client          string    `json:"observed_client,omitempty"`
	CallID          string    `json:"call_id,omitempty"`
	Status          string    `json:"status,omitempty"`
	Evidence        []string  `json:"evidence_refs,omitempty"`
	// Only an observer with evidence of credential transfer may populate this.
	DelegatedFrom string          `json:"delegated_from,omitempty"`
	Credentials   []CredentialUse `json:"credential_uses,omitempty"`
}

type CredentialUse struct {
	ID          string `json:"id"`
	Attribution string `json:"attribution_kind"`
}

func (e Event) Validate() error {
	if e.Version != Version {
		return errors.New("unsupported event schema")
	}
	if e.TS.IsZero() {
		return errors.New("missing event timestamp")
	}
	for _, field := range []struct{ name, value string }{{"event_id", e.ID}, {"source", e.Source}, {"session", e.Session}, {"span", e.Span}, {"agent", e.Agent}, {"kind", e.Kind}} {
		if field.value == "" || len(field.value) > 1024 || strings.ContainsAny(field.value, "\x00\r\n\t") {
			return fmt.Errorf("invalid %s", field.name)
		}
	}
	if len(e.Target) > 4096 || len(e.Evidence) > 64 {
		return errors.New("event metadata limit exceeded")
	}
	for _, v := range []string{e.Parent, e.ParentSession, e.RequestID, e.ParentRequestID, e.CallID, e.DelegatedFrom} {
		if len(v) > 1024 || strings.ContainsAny(v, "\x00\r\n\t") {
			return errors.New("invalid correlation metadata")
		}
	}
	for _, ref := range e.Evidence {
		if len(ref) > 1024 {
			return errors.New("evidence reference exceeds limit")
		}
	}
	if e.NHI != "" && !ValidIdentity(e.NHI) {
		return errors.New("invalid identity identifier")
	}
	if len(e.Credentials) > 64 {
		return errors.New("too many credential observations")
	}
	for _, c := range e.Credentials {
		if !ValidIdentity(c.ID) || (c.Attribution != "direct" && c.Attribution != "implicit") {
			return errors.New("invalid credential observation")
		}
	}
	switch e.Attribution {
	case "", "direct", "implicit", "unknown":
	default:
		return errors.New("invalid attribution kind")
	}
	return nil
}

var roleID = regexp.MustCompile(`^aws_role:arn:aws(?:-us-gov|-cn)?:((iam::[0-9]{12}:role/[A-Za-z0-9_+=,.@/-]+)|(sts::[0-9]{12}:assumed-role/[A-Za-z0-9_+=,.@/-]+/[A-Za-z0-9_+=,.@-]+))$`)

func ValidIdentity(id string) bool {
	if strings.HasPrefix(id, "aws_role:") {
		return len(id) < 2048 && roleID.MatchString(id)
	}
	if len(id) != 64 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

type Edge struct {
	NHI         string    `json:"nhi"`
	Agent       string    `json:"agent"`
	Path        []string  `json:"path"`
	Count       uint64    `json:"call_count"`
	First       time.Time `json:"first_seen"`
	Last        time.Time `json:"last_seen"`
	Attribution []string  `json:"attribution_kinds"`
	Evidence    []string  `json:"evidence_refs"`
	Unresolved  string    `json:"unresolved,omitempty"`
}

type Finding struct {
	WindowStart *time.Time `json:"observation_start,omitempty"`
	WindowEnd   *time.Time `json:"observation_end,omitempty"`
	Coverage    string     `json:"coverage"`
	ID          string     `json:"id"`
	Rule        string     `json:"rule"`
	Severity    string     `json:"severity"`
	NHI         string     `json:"nhi"`
	Agents      []string   `json:"agents,omitempty"`
	Reason      string     `json:"reason"`
	Remedy      string     `json:"remediation"`
	Evidence    []string   `json:"evidence_refs,omitempty"`
}

type Source struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type Manifest struct {
	Version   int       `json:"schema_version"`
	Operation string    `json:"operation"`
	RunID     string    `json:"run_id"`
	Started   time.Time `json:"started_at"`
	Ended     time.Time `json:"ended_at"`
	Complete  bool      `json:"complete"`
	Records   uint64    `json:"records"`
	Bytes     int64     `json:"bytes"`
	Errors    uint64    `json:"errors"`
	Skipped   uint64    `json:"skipped"`
	Sources   []Source  `json:"sources,omitempty"`
	Notes     []string  `json:"notes,omitempty"`
}
