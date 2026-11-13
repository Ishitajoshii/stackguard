package scout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

type API struct {
	client                *http.Client
	base, token, provider string
	limit                 int64
	queries               atomic.Int64
}

func NewGitHub(token string, limit int) *API {
	return newAPI("https://api.github.com", token, "github", limit)
}
func NewNotion(token string, limit int) *API {
	return newAPI("https://api.notion.com/v1", token, "notion", limit)
}
func newAPI(base, token, provider string, limit int) *API {
	return &API{base: base, token: token, provider: provider, limit: int64(limit), client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (a *API) Queries() int { return int(a.queries.Load()) }
func (a *API) Get(ctx context.Context, path string, q url.Values, out any) error {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "..") || strings.ContainsAny(path, "?#\r\n") {
		return errors.New("invalid API path")
	}
	if a.queries.Add(1) > a.limit {
		a.queries.Add(-1)
		return errors.New("API query budget exhausted; increase max_queries or narrow criteria")
	}
	r, e := http.NewRequestWithContext(ctx, "GET", a.base+path+"?"+q.Encode(), nil)
	if e != nil {
		return errors.New("invalid API request")
	}
	r.Header.Set("Accept", "application/vnd.github+json")
	r.Header.Set("User-Agent", "stackguard-oss-scout/1")
	if a.token != "" {
		r.Header.Set("Authorization", "Bearer "+a.token)
	}
	if a.provider == "github" {
		r.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	} else {
		r.Header.Set("Notion-Version", "2025-09-03")
		r.Header.Set("Accept", "application/json")
	}
	res, e := a.client.Do(r)
	if e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("provider unavailable; check connection and retry later")
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 401:
		return errors.New("provider authentication failed; refresh the dedicated token")
	case 403, 429:
		return errors.New("provider access denied or rate limited; check token scope and retry after the provider reset")
	case 404:
		return errors.New("provider resource missing or inaccessible; check link and sharing permissions")
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("provider request failed with HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
	if e != nil || len(b) > 2<<20 {
		return errors.New("provider response exceeds limit or is unreadable")
	}
	if json.Unmarshal(b, out) != nil {
		return errors.New("provider response schema is invalid")
	}
	return nil
}

type Repository struct {
	Name        string   `json:"full_name"`
	URL         string   `json:"html_url"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	Private     bool     `json:"private"`
	Archived    bool     `json:"archived"`
	Fork        bool     `json:"fork"`
	License     *struct {
		SPDX string `json:"spdx_id"`
	} `json:"license"`
	Pushed        time.Time `json:"pushed_at"`
	Updated       time.Time `json:"updated_at"`
	DefaultBranch string    `json:"default_branch"`
}
type Issue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"html_url"`
	State     string `json:"state"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PR      json.RawMessage `json:"pull_request"`
	Updated time.Time       `json:"updated_at"`
	Locked  bool            `json:"locked"`
}

var repoName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func (a *API) UserRepos(ctx context.Context, user string) ([]Repository, error) {
	if !handle.MatchString(user) {
		return nil, errors.New("invalid GitHub handle")
	}
	var repos []Repository
	e := a.Get(ctx, "/users/"+user+"/repos", url.Values{"per_page": {"100"}, "sort": {"updated"}, "type": {"owner"}}, &repos)
	return repos, e
}
func (a *API) Search(ctx context.Context, interest string, count int) ([]Repository, error) {
	if count < 1 || count > 100 || len(interest) > 100 {
		return nil, errors.New("invalid discovery bounds")
	}
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == ' ' {
			return r
		}
		return ' '
	}, interest)
	var v struct {
		Items      []Repository `json:"items"`
		Incomplete bool         `json:"incomplete_results"`
	}
	e := a.Get(ctx, "/search/repositories", url.Values{"q": {clean + " in:name,description,topics is:public archived:false"}, "sort": {"updated"}, "per_page": {fmt.Sprint(count)}}, &v)
	if e == nil && v.Incomplete {
		e = errors.New("GitHub search returned incomplete results; narrow interests")
	}
	return v.Items, e
}
func (a *API) Repo(ctx context.Context, repo string) (Repository, error) {
	var v Repository
	if !repoName.MatchString(repo) {
		return v, errors.New("invalid repository")
	}
	e := a.Get(ctx, "/repos/"+repo, nil, &v)
	return v, e
}
func (a *API) Issues(ctx context.Context, repo string, count int) ([]Issue, error) {
	if !repoName.MatchString(repo) || count < 1 || count > 100 {
		return nil, errors.New("invalid issue query")
	}
	var v []Issue
	e := a.Get(ctx, "/repos/"+repo+"/issues", url.Values{"state": {"open"}, "assignee": {"none"}, "per_page": {fmt.Sprint(count)}, "sort": {"updated"}}, &v)
	return v, e
}
func (a *API) Issue(ctx context.Context, repo string, n int) (Issue, error) {
	var v Issue
	if !repoName.MatchString(repo) || n < 1 {
		return v, errors.New("invalid issue")
	}
	e := a.Get(ctx, fmt.Sprintf("/repos/%s/issues/%d", repo, n), nil, &v)
	return v, e
}

// ActivePR returns uncertainty if the first bounded timeline page is full.
func (a *API) ActivePR(ctx context.Context, repo string, n int) (bool, bool, error) {
	if !repoName.MatchString(repo) || n < 1 {
		return false, false, errors.New("invalid issue")
	}
	var v []struct {
		Event  string `json:"event"`
		Source struct {
			Issue Issue `json:"issue"`
		} `json:"source"`
	}
	e := a.Get(ctx, fmt.Sprintf("/repos/%s/issues/%d/timeline", repo, n), url.Values{"per_page": {"100"}}, &v)
	if e != nil {
		return false, false, e
	}
	for _, x := range v {
		if x.Event == "cross-referenced" && len(x.Source.Issue.PR) > 0 && string(x.Source.Issue.PR) != "null" && x.Source.Issue.State == "open" {
			return true, true, nil
		}
	}
	return false, len(v) < 100, nil
}

type Note struct {
	Block string `json:"block_id"`
	Text  string `json:"text"`
}

func (a *API) Notes(ctx context.Context, page string) ([]Note, error) {
	root, e := NotionID(page)
	if e != nil || root == "" {
		return nil, errors.New("dedicated Notion page required")
	}
	if a.token == "" {
		return nil, errors.New("set STACKGUARD_NOTION_TOKEN for the dedicated shared page")
	}
	type node struct {
		id    string
		depth int
	}
	queue := []node{{root, 0}}
	seen := map[string]bool{}
	var notes []Note
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if seen[next.id] {
			continue
		}
		seen[next.id] = true
		cursor := ""
		for {
			var v struct {
				Results []map[string]json.RawMessage `json:"results"`
				More    bool                         `json:"has_more"`
				Cursor  string                       `json:"next_cursor"`
			}
			q := url.Values{"page_size": {"100"}}
			if cursor != "" {
				q.Set("start_cursor", cursor)
			}
			if e = a.Get(ctx, "/blocks/"+next.id+"/children", q, &v); e != nil {
				return notes, e
			}
			for _, b := range v.Results {
				if len(notes) >= 500 || len(queue) >= 500 {
					return notes, errors.New("Notion block limit reached; reduce dedicated-page content")
				}
				var id, kind string
				var children bool
				json.Unmarshal(b["id"], &id)
				json.Unmarshal(b["type"], &kind)
				json.Unmarshal(b["has_children"], &children)
				if kind == "child_page" || kind == "child_database" || kind == "link_to_page" || kind == "synced_block" {
					continue
				}
				var content struct {
					Text []struct {
						Plain string `json:"plain_text"`
					} `json:"rich_text"`
				}
				json.Unmarshal(b[kind], &content)
				var text strings.Builder
				for _, r := range content.Text {
					if text.Len()+len(r.Plain) > 8192 {
						return notes, errors.New("Notion block text limit reached")
					}
					text.WriteString(r.Plain)
				}
				notes = append(notes, Note{Block: id, Text: text.String()})
				if children {
					if next.depth >= 8 {
						return notes, errors.New("Notion depth limit reached")
					}
					child, e := NotionID(id)
					if e != nil {
						return notes, errors.New("invalid Notion child ID")
					}
					queue = append(queue, node{child, next.depth + 1})
				}
			}
			if !v.More {
				break
			}
			if v.Cursor == "" || v.Cursor == cursor {
				return notes, errors.New("invalid Notion pagination")
			}
			cursor = v.Cursor
		}
	}
	return notes, nil
}
