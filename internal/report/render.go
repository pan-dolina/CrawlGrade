package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Format is a report output format.
type Format string

// Supported report formats.
const (
	FormatTerminal Format = "terminal"
	FormatJSON     Format = "json"
	FormatHTML     Format = "html"
)

// ParseFormat parses a format name. It accepts the three supported formats
// case-insensitively and returns an error for anything else.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "terminal", "":
		return FormatTerminal, nil
	case "json":
		return FormatJSON, nil
	case "html":
		return FormatHTML, nil
	}
	return "", fmt.Errorf("unknown report format %q (want terminal, json or html)", s)
}

// Render writes the report in the given format to w.
func (rep *Report) Render(w io.Writer, format Format) error {
	switch format {
	case FormatJSON:
		out, err := RenderJSON(rep)
		if err != nil {
			return err
		}
		_, err = w.Write(out)
		return err
	case FormatHTML:
		return renderHTML(w, rep)
	case FormatTerminal:
		return RenderTerminal(w, rep)
	default:
		return fmt.Errorf("unknown report format %q", format)
	}
}

// CategoryGroups returns the group names that produced findings, in report
// order. It is used by callers that want to show which categories are present.
func (rep *Report) CategoryGroups() []string {
	var out []string
	for _, g := range orderedGroups {
		if len(rep.Groups[g]) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// findingsCategoryName returns the human-readable name of a group. It maps the
// group key to the findings category so the report and the JSON use the same
// labels.
func findingsCategoryName(group string) string {
	return findings.Category(group).Name()
}
