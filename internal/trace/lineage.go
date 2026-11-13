package trace

import (
	"fmt"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"io"
	"strings"
)

// Each bounded edge path is rendered as a tree; repeated roots retain distinct
// attribution/evidence rather than merging claims merely for presentation.
func writeLineage(w io.Writer, edge model.Edge) error {
	if _, err := fmt.Fprintf(w, "NHI %s\n", edge.NHI); err != nil {
		return err
	}
	for depth, agent := range edge.Path {
		if _, err := fmt.Fprintf(w, "%s`-- %q\n", strings.Repeat("    ", depth), agent); err != nil {
			return err
		}
	}
	indent := strings.Repeat("    ", len(edge.Path))
	if _, err := fmt.Fprintf(w, "%sagent=%q calls=%d attribution=%s first=%s last=%s\n", indent, edge.Agent, edge.Count, strings.Join(edge.Attribution, ","), edge.First.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), edge.Last.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")); err != nil {
		return err
	}
	if edge.Unresolved != "" {
		if _, err := fmt.Fprintf(w, "%sunresolved=%q\n", indent, edge.Unresolved); err != nil {
			return err
		}
	}
	for _, ref := range edge.Evidence {
		if _, err := fmt.Fprintf(w, "%sevidence=%q\n", indent, ref); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}
