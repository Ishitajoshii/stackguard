// Package secrets recognizes credential candidates and emits only fingerprints.
package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

type Match struct {
	Start, End           int
	Provider, ID, Masked string
}
type signature struct {
	provider string
	re       *regexp.Regexp
}

var signatures = []signature{
	{"github", regexp.MustCompile(`(?:gh[pousr]_[A-Za-z0-9]{20,255}|github_pat_[A-Za-z0-9_]{20,255})`)},
	{"notion", regexp.MustCompile(`(?:ntn_|secret_)[A-Za-z0-9]{20,255}`)},
	{"anthropic", regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,255}`)},
	{"openai", regexp.MustCompile(`sk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{20,255}`)},
	{"slack", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{16,255}`)},
	{"google", regexp.MustCompile(`AIza[A-Za-z0-9_-]{35}`)},
	{"aws", regexp.MustCompile(`(?:AKIA|ASIA)[A-Z0-9]{16}`)},
}
var sensitive = regexp.MustCompile(`(?i)(token|password|secret|authorization|api[_-]?key|access[_-]?key|credential)`)

func Fingerprint(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}
func Mask(value string) string {
	if len(value) <= 8 {
		return "[redacted]"
	}
	return value[:4] + "…" + value[len(value)-4:]
}
func Sensitive(name string) bool { return sensitive.MatchString(name) }
func Provider(name string) string {
	n := strings.ToLower(name)
	for _, p := range []string{"github", "notion", "anthropic", "azure", "openai", "slack", "google", "aws"} {
		if strings.Contains(n, p) {
			return p
		}
	}
	return "unknown"
}

// AWSSet fingerprints a length-delimited tuple as ONE credential. Missing pairs
// are not promoted to a usable credential. Role ARNs are separate principal IDs.
func AWSSet(access, secret, session string) (string, bool) {
	if access == "" || secret == "" {
		return "", false
	}
	b, _ := json.Marshal([]string{"aws-credential-set-v1", access, secret, session})
	return Fingerprint(string(b)), true
}
func AWSField(k string) bool {
	switch strings.ToUpper(k) {
	case "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN":
		return true
	}
	return false
}
func Kind(m Match) string {
	if m.Provider == "aws" && (strings.HasPrefix(m.Masked, "AKIA") || strings.HasPrefix(m.Masked, "ASIA")) {
		return "access_key_id"
	}
	return "secret"
}

// Detect returns candidates, not proof that a provider issued/accepted a key.
// Azure and other formatless keys require a sensitive provider-labelled field.
func Detect(text, field string) []Match {
	var result []Match
	for _, sig := range signatures {
		for _, pos := range sig.re.FindAllStringIndex(text, -1) {
			value := text[pos[0]:pos[1]]
			if sig.provider == "openai" && strings.HasPrefix(value, "sk-ant-") {
				continue
			}
			result = append(result, Match{pos[0], pos[1], sig.provider, Fingerprint(value), Mask(value)})
		}
	}
	if Sensitive(field) && len(text) >= 8 && len(text) <= 4096 && len(result) == 0 {
		value := strings.TrimSpace(strings.TrimPrefix(text, "Bearer "))
		if value != "" {
			result = append(result, Match{0, len(text), Provider(field), Fingerprint(value), Mask(value)})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Start < result[j].Start })
	return result
}

func Redact(text string) string {
	matches := Detect(text, "")
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		text = text[:m.Start] + "[credential:" + m.ID[:12] + "]" + text[m.End:]
	}
	return text
}

// JSON scrubs untrusted structured values before diagnostics/captures.
func JSON(v any, field string) any {
	if Sensitive(field) {
		return "[redacted]"
	}
	switch x := v.(type) {
	case string:
		return Redact(x)
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = JSON(v, field)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[Redact(k)] = JSON(v, k)
		}
		return out
	default:
		return v
	}
}
