// Package config parses explicitly selected files. It never discovers or reads
// an unrelated credential store. Raw values remain in memory only.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"github.com/pelletier/go-toml/v2"
	"sort"
	"strings"
)

type Server struct {
	Name       string
	Command    string            `toml:"command" json:"command"`
	Args       []string          `toml:"args" json:"args"`
	URL        string            `toml:"url" json:"url"`
	Env        map[string]string `toml:"env" json:"env"`
	EnvVars    []string          `toml:"env_vars" json:"env_vars"`
	Headers    map[string]string `toml:"http_headers" json:"http_headers"`
	HeaderRefs map[string]string `toml:"env_http_headers" json:"env_http_headers"`
	BearerRef  string            `toml:"bearer_token_env_var" json:"bearer_token_env_var"`
	CWD        string            `toml:"cwd" json:"cwd"`
}
type Document struct {
	Agent   string
	Servers []Server
	Role    string
}
type ConfigParser interface {
	Parse([]byte) (Document, error)
}
type Registry map[string]ConfigParser

func DefaultRegistry() Registry { return Registry{"codex": Codex{}, "agentcore": AgentCore{}} }

type Codex struct{}

func (Codex) Parse(b []byte) (Document, error) {
	var v struct {
		Servers map[string]Server `toml:"mcp_servers"`
	}
	if len(b) > 1<<20 {
		return Document{}, errors.New("config exceeds 1 MiB")
	}
	if err := toml.Unmarshal(b, &v); err != nil {
		return Document{}, errors.New("invalid Codex TOML")
	}
	d := Document{Agent: "codex"}
	for name, s := range v.Servers {
		s.Name = name
		d.Servers = append(d.Servers, s)
	}
	sort.Slice(d.Servers, func(i, j int) bool { return d.Servers[i].Name < d.Servers[j].Name })
	return d, nil
}

// AgentCore accepts a lab export, not an invented native credential file.
type AgentCore struct{}

func (AgentCore) Parse(b []byte) (Document, error) {
	var v struct {
		Schema  int      `json:"schema_version"`
		Role    string   `json:"execution_role"`
		Servers []Server `json:"servers"`
	}
	if len(b) > 1<<20 {
		return Document{}, errors.New("config exceeds 1 MiB")
	}
	if err := json.Unmarshal(b, &v); err != nil || v.Schema != 1 {
		return Document{}, errors.New("invalid versioned AgentCore lab export")
	}
	return Document{Agent: "agentcore", Servers: v.Servers, Role: v.Role}, nil
}
func Resolve(d Document, location string, env func(string) (string, bool)) []model.Identity {
	var out []model.Identity
	add := func(value, ref, field, server string) {
		b := model.Binding{Agent: d.Agent, Server: server, Location: secrets.Redact(location + ":" + field), Reference: ref}
		if ref != "" {
			var ok bool
			value, ok = env(ref)
			if !ok || value == "" {
				out = append(out, model.Identity{Version: 1, ID: secrets.Fingerprint("unresolved:" + d.Agent + ":" + ref), Provider: secrets.Provider(ref), Kind: "reference", Unresolved: true, Bindings: []model.Binding{b}})
				return
			}
		}
		value = strings.TrimPrefix(value, "Bearer ")
		matches := secrets.Detect(value, field+" "+ref)
		for _, m := range matches {
			out = append(out, model.Identity{Version: 1, ID: m.ID, Provider: m.Provider, Kind: secrets.Kind(m), Masked: m.Masked, Bindings: []model.Binding{b}})
		}
	}
	for _, s := range d.Servers {
		awsValues := map[string]string{}
		awsBindings := []model.Binding{}
		for _, k := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
			if v, ok := s.Env[k]; ok {
				awsValues[k] = v
				awsBindings = append(awsBindings, model.Binding{Agent: d.Agent, Server: s.Name, Location: location + ":env." + k})
			} else {
				for _, ref := range s.EnvVars {
					if ref == k {
						if v, ok := env(ref); ok {
							awsValues[k] = v
							awsBindings = append(awsBindings, model.Binding{Agent: d.Agent, Server: s.Name, Location: location + ":env_vars", Reference: ref})
						}
					}
				}
			}
		}
		awsID, awsComplete := secrets.AWSSet(awsValues["AWS_ACCESS_KEY_ID"], awsValues["AWS_SECRET_ACCESS_KEY"], awsValues["AWS_SESSION_TOKEN"])
		if awsComplete {
			out = append(out, model.Identity{Version: 1, ID: awsID, Provider: "aws", Kind: "credential_set", Bindings: awsBindings})
		}
		for k, v := range s.Env {
			if awsComplete && secrets.AWSField(k) {
				continue
			}
			add(v, "", k, s.Name)
		}
		for _, k := range s.EnvVars {
			if awsComplete && secrets.AWSField(k) {
				continue
			}
			add("", k, k, s.Name)
		}
		for k, v := range s.Headers {
			add(v, "", k, s.Name)
		}
		for k, v := range s.HeaderRefs {
			add("", v, k, s.Name)
		}
		if s.BearerRef != "" {
			add("", s.BearerRef, "authorization", s.Name)
		}
		for i, v := range s.Args {
			add(v, "", fmt.Sprintf("args[%d]", i), s.Name)
		}
		add(s.URL, "", "url", s.Name)
	}
	if d.Role != "" {
		if model.ValidIdentity("aws_role:" + d.Role) {
			out = append(out, model.Identity{Version: 1, ID: "aws_role:" + d.Role, Provider: "aws", Kind: "principal", Bindings: []model.Binding{{Agent: d.Agent, Server: "execution", Location: location + ":execution_role"}}})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			return out[i].Bindings[0].Location < out[j].Bindings[0].Location
		}
		return out[i].ID < out[j].ID
	})
	return out
}
