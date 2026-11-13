package scout

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/output"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func WriteReport(dir string, p Profile, r Research, session string) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	clean := func(v any) (any, error) {
		b, e := json.Marshal(v)
		if e != nil {
			return nil, e
		}
		var result any
		e = json.Unmarshal([]byte(secrets.Redact(string(b))), &result)
		return result, e
	}
	safe, e := clean(map[string]any{"schema_version": 1, "session": session, "research": r})
	if e != nil {
		return e
	}
	if e = output.Publish(filepath.Join(dir, "recommendations.json"), safe); e != nil {
		return e
	}
	var md strings.Builder
	fmt.Fprintf(&md, "# OSS contribution shortlist\n\nSession: `%s`\n\n", session)
	if !r.Complete {
		md.WriteString("**Partial research:** some source checks failed or exhausted the query budget.\n\n")
	}
	for _, v := range r.Recommendations {
		title := strings.NewReplacer("[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ").Replace(v.Title)
		fmt.Fprintf(&md, "## %s — #%d\n\n[%s](%s) · [Repository](%s)\n\nScore: %d/100. Estimated effort: %d–%d hours. Rechecked: %s.\n\n", v.Repository, v.Issue, title, v.IssueURL, v.RepositoryURL, v.Score, v.EffortHours[0], v.EffortHours[1], v.Checked.Format(time.RFC3339))
		for _, e := range v.Evidence {
			fmt.Fprintf(&md, "- Evidence: %s\n", e)
		}
		for _, e := range v.Uncertainty {
			fmt.Fprintf(&md, "- Uncertainty: %s\n", e)
		}
		md.WriteString("\nFirst steps:\n\n")
		for n, s := range v.FirstSteps {
			fmt.Fprintf(&md, "%d. %s\n", n+1, s)
		}
		if v.Draft != "" {
			fmt.Fprintf(&md, "\nLocal draft only (not posted):\n\n> %s\n", v.Draft)
		}
		md.WriteString("\n")
	}
	md.WriteString("## Coverage\n\n")
	for _, note := range r.Coverage {
		fmt.Fprintf(&md, "- %s\n", note)
	}
	f, e := os.OpenFile(filepath.Join(dir, "report.md"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.WriteString(secrets.Redact(md.String()))
	e = errors.Join(e, f.Close())
	if e != nil {
		return e
	}
	profile := map[string]any{"github": p.GitHub, "interests": p.Interests, "languages": p.Languages, "effort": p.Effort, "hours_budget": p.Hours, "activity_days": p.ActivityDays, "candidates": p.Candidates, "issues_per_repository": p.Issues, "results": p.Results, "resume_configured": p.Resume != "", "notion_configured": p.Notion != ""}
	manifest := map[string]any{"schema_version": 1, "session_id": session, "complete": r.Complete, "checked_at": r.Checked, "ended_at": time.Now().UTC(), "query_count": r.Queries, "effective_profile": profile, "source_coverage": r.Coverage, "recommendation_count": len(r.Recommendations)}
	safe, e = clean(manifest)
	if e != nil {
		return e
	}
	return output.Publish(filepath.Join(dir, "manifest.json"), safe)
}
