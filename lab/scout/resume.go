package scout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SkillEvidence struct {
	Skill   string   `json:"skill"`
	Kind    string   `json:"kind"`
	Sources []string `json:"sources"`
}
type Summary struct {
	Version  int             `json:"schema_version"`
	Skills   []SkillEvidence `json:"skills"`
	Coverage []string        `json:"coverage"`
	Reviewed bool            `json:"reviewed"`
}
type page struct {
	Number int    `json:"page"`
	Text   string `json:"text"`
}

func ExtractSummary(ctx context.Context, p Profile, python, extractor, temp string) (Summary, error) {
	s := Summary{Version: 1, Coverage: []string{"Skill mentions are self-reported evidence, not proficiency verification."}}
	if p.Resume == "" {
		s.Coverage = append(s.Coverage, "Resume skipped; reduced profile coverage.")
		return s, nil
	}
	path := strings.Trim(p.Resume, "\"")
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() {
		return s, errors.New("resume path must be a readable local PDF, .txt or .md file")
	}
	if info.Size() > 20<<20 {
		return s, errors.New("resume exceeds 20 MiB; choose a smaller export or skip")
	}
	var pages []page
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md":
		b, e := os.ReadFile(path)
		if e != nil {
			return s, errors.New("resume cannot be read")
		}
		pages = []page{{1, string(b)}}
	case ".pdf":
		dir, e := os.MkdirTemp(temp, "stackguard-resume-*")
		if e != nil {
			return s, e
		}
		defer os.RemoveAll(dir)
		dest := filepath.Join(dir, "pages.json")
		deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		command := exec.CommandContext(deadline, python, extractor, path, "--output", dest)
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if e = command.Run(); e != nil {
			if ctx.Err() != nil {
				return s, ctx.Err()
			}
			return s, errors.New("PDF extraction failed: check Python/pypdf; scanned, encrypted or corrupt PDFs need a text export, .txt/.md fallback or resume skip")
		}
		f, e := os.Open(dest)
		if e != nil {
			return s, e
		}
		b, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
		f.Close()
		if e != nil || len(b) > 4<<20 {
			return s, errors.New("extracted resume exceeds 4 MiB")
		}
		var v struct {
			Pages []page `json:"pages"`
		}
		if json.Unmarshal(b, &v) != nil {
			return s, errors.New("invalid extractor output")
		}
		pages = v.Pages
	default:
		return s, errors.New("use PDF, .txt or .md; OCR is not supported")
	}
	if len(pages) == 0 {
		return s, errors.New("resume has no extractable pages; choose another file or skip")
	}
	keywords := []string{"Go", "Python", "JavaScript", "TypeScript", "Java", "Rust", "C++", "SQL", "PostgreSQL", "Redis", "Docker", "Kubernetes", "AWS", "Linux", "Git", "REST", "GraphQL", "React", "FastAPI", "Django", "Node.js", "MCP", "LLM", "agents", "backend", "testing", "CI/CD"}
	keywords = append(keywords, p.Languages...)
	found := map[string][]string{}
	characters := 0
	for _, pg := range pages {
		characters += len(strings.TrimSpace(pg.Text))
		for _, word := range keywords {
			re := regexp.MustCompile(`(?i)(?:^|[^a-z0-9])` + regexp.QuoteMeta(word) + `(?:$|[^a-z0-9])`)
			if re.MatchString(pg.Text) {
				found[word] = append(found[word], fmt.Sprintf("resume:page:%d", pg.Number))
			}
		}
	}
	if characters == 0 {
		return s, errors.New("resume is empty; choose another file or skip")
	}
	for word, refs := range found {
		s.Skills = append(s.Skills, SkillEvidence{Skill: word, Kind: "self_reported_mention", Sources: refs})
	}
	sort.Slice(s.Skills, func(i, j int) bool { return s.Skills[i].Skill < s.Skills[j].Skill })
	if len(s.Skills) == 0 {
		s.Coverage = append(s.Coverage, "No recognized skill mentions; edit the summary with source references before review.")
	}
	return s, nil
}
func LoadSummary(path string) (Summary, error) {
	var s Summary
	f, e := os.Open(path)
	if e != nil {
		return s, errors.New("reviewed summary not found")
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if e != nil || len(b) > 64<<10 || json.Unmarshal(b, &s) != nil || s.Version != 1 {
		return s, errors.New("invalid summary")
	}
	if !s.Reviewed {
		return s, errors.New("review summary.json and set reviewed to true before cloud use")
	}
	if len(s.Skills) > 128 {
		return s, errors.New("too many summary skills")
	}
	for _, skill := range s.Skills {
		if len(skill.Skill) > 100 || len(skill.Sources) > 32 {
			return s, errors.New("summary limits exceeded")
		}
	}
	return s, nil
}
