// Package scout implements the lab application separately from probe analysis.
package scout

import (
	"bufio"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Profile struct {
	Present      map[string]bool `yaml:"-" json:"-"`
	Version      int             `yaml:"schema_version" json:"schema_version"`
	GitHub       string          `yaml:"github" json:"github"`
	Resume       string          `yaml:"resume,omitempty" json:"resume,omitempty"`
	Interests    []string        `yaml:"interests" json:"interests"`
	Languages    []string        `yaml:"languages,omitempty" json:"languages,omitempty"`
	Effort       string          `yaml:"effort" json:"effort"`
	Notion       string          `yaml:"notion_page,omitempty" json:"notion_page,omitempty"`
	Hours        int             `yaml:"hours_budget" json:"hours_budget"`
	Exclude      []string        `yaml:"exclude_repositories,omitempty" json:"exclude_repositories,omitempty"`
	Types        []string        `yaml:"contribution_types,omitempty" json:"contribution_types,omitempty"`
	ActivityDays int             `yaml:"activity_days" json:"activity_days"`
	Candidates   int             `yaml:"candidates" json:"candidates"`
	Issues       int             `yaml:"issues_per_repository" json:"issues_per_repository"`
	Results      int             `yaml:"results" json:"results"`
	MaxQueries   int             `yaml:"max_queries" json:"max_queries"`
	AWSProfile   string          `yaml:"aws_profile,omitempty" json:"aws_profile,omitempty"`
	Region       string          `yaml:"aws_region,omitempty" json:"aws_region,omitempty"`
	Harness      string          `yaml:"harness_arn,omitempty" json:"harness_arn,omitempty"`
	Unknown      map[string]any  `yaml:",inline" json:"-"`
}

func Defaults() Profile {
	return Profile{Version: 1, Interests: []string{"ai-agent-tools", "backend"}, Effort: "meaningful-starter", Hours: 12, ActivityDays: 90, Candidates: 20, Issues: 5, Results: 5, MaxQueries: 160, AWSProfile: "default"}
}

var handle = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
var pageID = regexp.MustCompile(`(?i)[a-f0-9]{32}$`)

// GitHubHandle accepts a profile link or handle, never a repository URL.
func GitHubHandle(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "https://") {
		u, e := url.Parse(s)
		if e != nil || u.User != nil || (u.Host != "github.com" && u.Host != "www.github.com") {
			return "", errors.New("use a GitHub profile URL or handle")
		}
		s = strings.Trim(u.Path, "/")
	}
	if !handle.MatchString(s) {
		return "", errors.New("use a GitHub profile URL or handle")
	}
	return s, nil
}

func NotionID(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	u, e := url.Parse(s)
	if e == nil && u.Host != "" {
		if u.Scheme != "https" || !(u.Hostname() == "notion.so" || strings.HasSuffix(u.Hostname(), ".notion.so") || u.Hostname() == "notion.com" || strings.HasSuffix(u.Hostname(), ".notion.com")) {
			return "", errors.New("Notion URL must use notion.so or notion.com")
		}
		s = strings.TrimSuffix(u.Path, "/")
	}
	s = strings.ReplaceAll(s, "-", "")
	id := pageID.FindString(s)
	if id == "" {
		return "", errors.New("Notion page needs a valid page ID or link")
	}
	return strings.ToLower(id), nil
}
func (p Profile) Validate() error {
	if p.Version != 1 {
		return errors.New("unsupported profile version")
	}
	if !handle.MatchString(p.GitHub) {
		return errors.New("enter a valid GitHub handle")
	}
	if len(p.Interests) == 0 || len(p.Interests) > 12 {
		return errors.New("provide 1..12 interests")
	}
	for _, group := range [][]string{p.Interests, p.Languages, p.Types, p.Exclude} {
		if len(group) > 32 {
			return errors.New("too many profile criteria")
		}
		for _, s := range group {
			if len(s) > 100 || strings.ContainsAny(s, "\r\n\x00") {
				return errors.New("invalid profile criterion")
			}
		}
	}
	switch p.Effort {
	case "tiny", "meaningful-starter", "stretch":
	default:
		return errors.New("effort must be tiny, meaningful-starter or stretch")
	}
	if p.Hours < 1 || p.Hours > 160 || p.ActivityDays < 1 || p.ActivityDays > 365 || p.Candidates < 1 || p.Candidates > 50 || p.Issues < 1 || p.Issues > 20 || p.Results < 1 || p.Results > 10 || p.MaxQueries < 1 || p.MaxQueries > 1000 {
		return errors.New("advanced profile limits are out of range")
	}
	_, e := NotionID(p.Notion)
	return e
}
func Load(path string) (Profile, error) {
	p := Defaults()
	f, e := os.Open(path)
	if e != nil {
		return p, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(b) > 1<<20 {
		return p, errors.New("profile exceeds 1 MiB or cannot be read")
	}
	if yaml.Unmarshal(b, &p) != nil {
		return p, errors.New("invalid YAML profile")
	}
	p.GitHub, e = GitHubHandle(p.GitHub)
	if e != nil {
		return p, e
	}
	var fields map[string]yaml.Node
	_ = yaml.Unmarshal(b, &fields)
	p.Present = map[string]bool{}
	for key := range fields {
		p.Present[key] = true
	}
	return p, p.Validate()
}
func Save(path string, p Profile) error {
	if e := p.Validate(); e != nil {
		return e
	}
	b, e := yaml.Marshal(p)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".profile-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(name, path)
}
func list(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func Questionnaire(in io.Reader, out io.Writer, p Profile) (Profile, error) {
	r := bufio.NewReader(in)
	labels := []string{"GitHub profile URL or handle", "Resume PDF/text path (skip to omit)", "Interests (comma separated)", "Languages (none for no filter)", "Effort (tiny / meaningful-starter / stretch)", "Notion page link (skip to omit)"}
	ask := func(i int) error {
		values := []string{p.GitHub, p.Resume, strings.Join(p.Interests, ","), strings.Join(p.Languages, ","), p.Effort, p.Notion}
		fmt.Fprintf(out, "%d. %s [%s]: ", i+1, labels[i], values[i])
		s, e := r.ReadString('\n')
		if e != nil {
			return errors.New("setup needs interactive answers; use --profile for unattended runs")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		switch i {
		case 0:
			p.GitHub = s
		case 1:
			p.Resume = strings.Trim(s, "\"")
			if s == "skip" {
				p.Resume = ""
			}
		case 2:
			p.Interests = list(s)
		case 3:
			p.Languages = list(s)
			if s == "none" {
				p.Languages = nil
			}
		case 4:
			p.Effort = s
		case 5:
			p.Notion = s
			if s == "skip" {
				p.Notion = ""
			}
		}
		return nil
	}
	for i := range labels {
		if e := ask(i); e != nil {
			return p, e
		}
	}
	for {
		fmt.Fprintf(out, "\nProfile: GitHub=%s; resume=%t; interests=%s; languages=%s; effort=%s; Notion=%t\nSave? [yes / question number 1-6]: ", p.GitHub, p.Resume != "", strings.Join(p.Interests, ","), strings.Join(p.Languages, ","), p.Effort, p.Notion != "")
		s, e := r.ReadString('\n')
		if e != nil {
			return p, errors.New("setup cancelled before confirmation")
		}
		s = strings.TrimSpace(s)
		if s == "yes" || s == "y" {
			normalized, err := GitHubHandle(p.GitHub)
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			p.GitHub = normalized
			if e = p.Validate(); e != nil {
				fmt.Fprintln(out, e)
				continue
			}
			return p, nil
		}
		if len(s) == 1 && s[0] >= '1' && s[0] <= '6' {
			if e = ask(int(s[0] - '1')); e != nil {
				return p, e
			}
		} else {
			return p, errors.New("profile not saved")
		}
	}
}
