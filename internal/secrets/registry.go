package secrets

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

type SecretDetector interface {
	Name() string
	Detect(text, field string) []Match
}
type Registry struct{ detectors map[string]SecretDetector }

func NewRegistry() *Registry { return &Registry{detectors: map[string]SecretDetector{}} }
func (r *Registry) Register(d SecretDetector) error {
	if d == nil || d.Name() == "" {
		return errors.New("detector name required")
	}
	if _, ok := r.detectors[d.Name()]; ok {
		return errors.New("detector already registered")
	}
	r.detectors[d.Name()] = d
	return nil
}
func (r *Registry) Detect(text, field string) []Match {
	names := make([]string, 0, len(r.detectors))
	for name := range r.detectors {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Match
	for _, name := range names {
		out = append(out, r.detectors[name].Detect(text, field)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

type providerDetector struct {
	name string
	re   *regexp.Regexp
}

func (d providerDetector) Name() string { return d.name }
func (d providerDetector) Detect(text, field string) []Match {
	var out []Match
	if d.re != nil {
		for _, pos := range d.re.FindAllStringIndex(text, -1) {
			value := text[pos[0]:pos[1]]
			if d.name == "openai" && strings.HasPrefix(value, "sk-ant-") {
				continue
			}
			out = append(out, Match{pos[0], pos[1], d.name, Fingerprint(value), Mask(value)})
		}
	}
	if len(out) == 0 && Sensitive(field) && Provider(field) == d.name && len(text) >= 8 && len(text) <= 4096 {
		v := strings.TrimSpace(strings.TrimPrefix(text, "Bearer "))
		out = append(out, Match{0, len(text), d.name, Fingerprint(v), Mask(v)})
	}
	return out
}
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, s := range signatures {
		_ = r.Register(providerDetector{s.provider, s.re})
	}
	_ = r.Register(providerDetector{name: "azure"})
	return r
}

// Contextual Azure keys have no universal prefix. This detects only explicit
// Azure key assignments; a match is a candidate, never proof of validity.
var azureAssignment = regexp.MustCompile(`(?i)(?:azure[_a-z0-9-]{0,64}(?:key|token)|AZURE_OPENAI_API_KEY)["' \t]*[:=][ \t"']*([A-Za-z0-9_+/=-]{16,256})`)

func contextualAzure(text string) []Match {
	var out []Match
	for _, p := range azureAssignment.FindAllStringSubmatchIndex(text, -1) {
		value := text[p[2]:p[3]]
		out = append(out, Match{p[2], p[3], "azure", Fingerprint(value), Mask(value)})
	}
	return out
}
