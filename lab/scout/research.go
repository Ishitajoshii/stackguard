package scout

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Recommendation struct {
	Repository    string         `json:"repository"`
	RepositoryURL string         `json:"repository_url"`
	Issue         int            `json:"issue"`
	IssueURL      string         `json:"issue_url"`
	Title         string         `json:"title"`
	Score         int            `json:"score"`
	Components    map[string]int `json:"components"`
	Evidence      []string       `json:"evidence"`
	Uncertainty   []string       `json:"uncertainty"`
	EffortHours   [2]int         `json:"effort_hours"`
	FirstSteps    []string       `json:"first_steps"`
	Draft         string         `json:"draft_comment,omitempty"`
	Checked       time.Time      `json:"checked_at"`
}
type Research struct {
	Recommendations []Recommendation `json:"recommendations"`
	Coverage        []string         `json:"coverage"`
	Queries         int              `json:"query_count"`
	Complete        bool             `json:"complete"`
	Checked         time.Time        `json:"checked_at"`
}

func Suitable(repo Repository, issue Issue) bool {
	return !repo.Private && !repo.Archived && repo.License != nil && repo.License.SPDX != "" && repo.License.SPDX != "NOASSERTION" && issue.State == "open" && len(issue.Assignees) == 0 && !issue.Locked && (len(issue.PR) == 0 || string(issue.PR) == "null") && issue.Number > 0
}
func Score(p Profile, s Summary, r Repository, i Issue, now time.Time, maintainer bool, onboarding string) Recommendation {
	v := Recommendation{Repository: r.Name, RepositoryURL: "https://github.com/" + r.Name, Issue: i.Number, IssueURL: fmt.Sprintf("https://github.com/%s/issues/%d", r.Name, i.Number), Title: i.Title, Components: map[string]int{}, Checked: now, Uncertainty: []string{"Effort is a heuristic range; inspect code and ask the maintainer before committing.", "Unassigned does not prove unclaimed; read recent comments."}}
	v.Components["skill_fit"] = 0
	for _, skill := range s.Skills {
		if strings.EqualFold(skill.Skill, r.Language) {
			v.Components["skill_fit"] = 30
			v.Evidence = append(v.Evidence, "Language match: "+skill.Skill+" ("+skill.Kind+"; "+strings.Join(skill.Sources, ",")+"). Repository primary language is not proficiency proof.")
			break
		}
	}
	text := strings.ToLower(r.Name + " " + r.Description + " " + strings.Join(r.Topics, " "))
	for _, interest := range p.Interests {
		for _, word := range strings.Fields(strings.ReplaceAll(interest, "-", " ")) {
			if len(word) > 2 && strings.Contains(text, strings.ToLower(word)) {
				v.Components["interest_fit"] = 25
				v.Evidence = append(v.Evidence, "Interest keyword match in repository metadata: "+word)
				break
			}
		}
	}
	v.Components["feasibility"] = 5
	for _, l := range i.Labels {
		label := strings.ToLower(l.Name)
		if strings.Contains(label, "good first") || strings.Contains(label, "beginner") {
			v.Components["feasibility"] = 20
		} else if strings.Contains(label, "help wanted") && v.Components["feasibility"] < 15 {
			v.Components["feasibility"] = 15
		}
	}
	switch p.Effort {
	case "tiny":
		v.EffortHours = [2]int{1, 3}
	case "stretch":
		v.EffortHours = [2]int{12, 40}
	default:
		v.EffortHours = [2]int{4, 12}
	}
	if len(i.Body) < 80 {
		v.Uncertainty = append(v.Uncertainty, "Issue description is sparse; clarify acceptance criteria.")
		v.Components["feasibility"] = 0
	}
	if v.EffortHours[1] > p.Hours {
		v.Uncertainty = append(v.Uncertainty, "Upper effort estimate exceeds your hours budget.")
	}
	if maintainer {
		v.Components["maintainer_activity"] = 15
		v.Evidence = append(v.Evidence, "Recent issue comment by OWNER/MEMBER/COLLABORATOR within configured activity window.")
	} else if !r.Pushed.IsZero() && r.Pushed.After(now.AddDate(0, 0, -p.ActivityDays)) {
		v.Components["maintainer_activity"] = 7
		v.Evidence = append(v.Evidence, "Repository push at "+r.Pushed.UTC().Format(time.RFC3339)+"; activity proxy, not maintainer responsiveness.")
		v.Uncertainty = append(v.Uncertainty, "Recent maintainer response not established.")
	} else {
		v.Components["maintainer_activity"] = 0
		v.Uncertainty = append(v.Uncertainty, "Activity evidence unknown or outside the window; project is not labelled inactive.")
	}
	if onboarding != "" {
		v.Components["onboarding"] = 10
		v.Evidence = append(v.Evidence, "Contribution guide: "+onboarding)
	} else {
		v.Components["onboarding"] = 0
		v.Uncertainty = append(v.Uncertainty, "Contribution guide not established.")
	}
	for _, n := range v.Components {
		v.Score += n
	}
	v.FirstSteps = []string{"Read the issue, recent discussion and contribution guide; confirm it is available.", "Clone the repository and reproduce the issue with its documented test command.", "Propose a narrowly scoped change and add a regression test before opening a PR."}
	return v
}
func (a *API) MaintainerActive(ctx context.Context, repo string, n int, since time.Time) (bool, error) {
	var comments []struct {
		Association string    `json:"author_association"`
		Created     time.Time `json:"created_at"`
	}
	e := a.Get(ctx, fmt.Sprintf("/repos/%s/issues/%d/comments", repo, n), url.Values{"per_page": {"100"}, "since": {since.UTC().Format(time.RFC3339)}}, &comments)
	if e != nil {
		return false, e
	}
	for _, c := range comments {
		if c.Created.After(since) && (c.Association == "OWNER" || c.Association == "MEMBER" || c.Association == "COLLABORATOR") {
			return true, nil
		}
	}
	return false, nil
}
func (a *API) Onboarding(ctx context.Context, repo string) (string, error) {
	var v struct {
		Files struct {
			Contributing *struct {
				URL string `json:"html_url"`
			} `json:"contributing"`
		} `json:"files"`
	}
	e := a.Get(ctx, "/repos/"+repo+"/community/profile", nil, &v)
	if e != nil {
		return "", e
	}
	if v.Files.Contributing != nil {
		return v.Files.Contributing.URL, nil
	}
	return "", nil
}
func Discover(ctx context.Context, p Profile, s Summary, api *API, draft bool) (Research, error) {
	result := Research{Checked: time.Now().UTC(), Complete: true, Coverage: []string{"Heuristic scores use 30/25/20/15/10 weights; they are reproducible for frozen evidence, not a model's objective assessment."}}
	if e := p.Validate(); e != nil {
		return result, e
	}
	if !s.Reviewed {
		return result, errors.New("review the normalized summary before research")
	}
	repos := map[string]Repository{}
	excluded := map[string]bool{}
	for _, r := range p.Exclude {
		excluded[strings.ToLower(r)] = true
	}
	for _, interest := range p.Interests {
		items, e := api.Search(ctx, interest, p.Candidates)
		if e != nil {
			result.Complete = false
			result.Coverage = append(result.Coverage, e.Error())
			break
		}
		for _, r := range items {
			if len(repos) >= p.Candidates {
				break
			}
			if !excluded[strings.ToLower(r.Name)] && repoName.MatchString(r.Name) {
				repos[r.Name] = r
			}
		}
		if len(repos) >= p.Candidates {
			break
		}
	}
	names := make([]string, 0, len(repos))
	for name := range repos {
		names = append(names, name)
	}
	sort.Strings(names)
	var candidates []Recommendation
	for _, name := range names {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		r := repos[name]
		if r.Private || r.Archived || r.License == nil {
			continue
		}
		if len(p.Languages) > 0 {
			match := false
			for _, l := range p.Languages {
				if strings.EqualFold(l, r.Language) {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		issues, e := api.Issues(ctx, name, p.Issues)
		if e != nil {
			result.Complete = false
			result.Coverage = append(result.Coverage, e.Error())
			break
		}
		guide, guideErr := api.Onboarding(ctx, name)
		if guideErr != nil {
			guide = ""
		}
		for _, issue := range issues {
			if !Suitable(r, issue) {
				continue
			}
			active, complete, e := api.ActivePR(ctx, name, issue.Number)
			if e != nil {
				result.Complete = false
				result.Coverage = append(result.Coverage, e.Error())
				break
			}
			if active {
				continue
			}
			maintainer, e := api.MaintainerActive(ctx, name, issue.Number, result.Checked.AddDate(0, 0, -p.ActivityDays))
			v := Score(p, s, r, issue, result.Checked, maintainer && e == nil, guide)
			if v.EffortHours[0] > p.Hours || v.Components["feasibility"] == 0 {
				continue
			}
			if !complete {
				v.Uncertainty = append(v.Uncertainty, "Timeline exceeded bounded page; PR linkage remains uncertain.")
			}
			if e != nil {
				v.Uncertainty = append(v.Uncertainty, "Maintainer comment check failed; activity coverage reduced.")
			}
			if len(p.Types) > 0 {
				matches := false
				body := strings.ToLower(issue.Title + " " + issue.Body)
				for _, kind := range p.Types {
					if strings.Contains(body, strings.ToLower(kind)) {
						matches = true
					}
				}
				if !matches {
					continue
				}
			}
			candidates = append(candidates, v)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Repository != b.Repository {
			return a.Repository < b.Repository
		}
		return a.Issue < b.Issue
	})
	selected := map[string]bool{}
	for _, v := range candidates {
		if len(result.Recommendations) >= p.Results {
			break
		}
		if selected[v.Repository] {
			continue
		}
		latest, e := api.Issue(ctx, v.Repository, v.Issue)
		if e != nil {
			result.Complete = false
			result.Coverage = append(result.Coverage, "Final issue recheck failed: "+e.Error())
			break
		}
		if !Suitable(repos[v.Repository], latest) {
			continue
		}
		active, complete, e := api.ActivePR(ctx, v.Repository, v.Issue)
		if e != nil {
			result.Complete = false
			result.Coverage = append(result.Coverage, "Final PR linkage check failed: "+e.Error())
			break
		}
		if active {
			continue
		}
		if !complete {
			v.Uncertainty = append(v.Uncertainty, "Final timeline scan is partial.")
		}
		v.Checked = time.Now().UTC()
		if draft {
			v.Draft = fmt.Sprintf("Hi! I am interested in working on #%d. Is this still available, and are there acceptance criteria or a suggested starting point I should check first?", v.Issue)
		}
		selected[v.Repository] = true
		result.Recommendations = append(result.Recommendations, v)
	}
	result.Queries = api.Queries()
	if len(result.Recommendations) < p.Results {
		result.Coverage = append(result.Coverage, fmt.Sprintf("Returned %d/%d requested projects; no filters were loosened. Sparse matches, ownership, query limits or final-state changes can reduce results.", len(result.Recommendations), p.Results))
	}
	return result, nil
}
