package scout

import (
	"strconv"
	"strings"
)

// ApplyNotes understands only four plain preference keys. It never executes or
// follows instructions/URLs in Notion. Explicit saved/CLI fields win.
func ApplyNotes(p Profile, notes []Note, explicit map[string]bool) Profile {
	for _, note := range notes {
		for _, line := range strings.Split(note.Text, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key = strings.ToLower(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			if explicit[key] {
				continue
			}
			next := p
			switch key {
			case "interests":
				next.Interests = list(value)
			case "languages":
				next.Languages = list(value)
				if value == "none" {
					next.Languages = nil
				}
			case "effort":
				next.Effort = value
			case "hours_budget":
				n, e := strconv.Atoi(value)
				if e != nil {
					continue
				}
				next.Hours = n
			default:
				continue
			}
			if next.Validate() == nil {
				p = next
			}
		}
	}
	return p
}
