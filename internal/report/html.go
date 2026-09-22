package report

import (
	"io"
	"sort"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// renderHTML writes a standalone HTML report to w. The document is fully
// self-contained: it embeds its styling, loads nothing from the network, and
// contains no script derived from the audited site. All dynamic values are
// passed through html/template, which escapes them for the context they are
// rendered in.
func renderHTML(w io.Writer, rep *Report) error { return renderHTMLMode(w, rep, false) }

// RenderHTMLQuickWins writes a focused standalone report containing the
// summary and a short prioritized action list.
func RenderHTMLQuickWins(w io.Writer, rep *Report) error { return renderHTMLMode(w, rep, true) }

func renderHTMLMode(w io.Writer, rep *Report, quickOnly bool) error {
	data := htmlModel{
		Report:    rep,
		StartURL:  rep.Summary.StartURL,
		Pages:     rep.Summary.Pages,
		Counts:    summaryCounts(rep.Summary.Findings),
		Highest:   rep.Summary.Highest,
		Stop:      rep.Summary.StopReason,
		Skipped:   skippedRows(rep.Summary.Skipped),
		Groups:    reportGroups(rep),
		QuickWins: quickWins(rep),
		QuickOnly: quickOnly,
	}
	t, err := loadHTMLTemplate()
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

// htmlModel is the template model. Every string field is escaped by
// html/template for the context it is rendered in.
type htmlModel struct {
	Report    *Report
	StartURL  string
	Pages     int
	Counts    severityCounts
	Highest   string
	Stop      string
	Skipped   []skipRow
	Groups    []groupModel
	QuickWins []findingModel
	QuickOnly bool
}

func quickWins(rep *Report) []findingModel {
	var out []findingModel
	seen := map[string]bool{}
	for _, sev := range []findings.Severity{findings.SeverityCritical, findings.SeverityHigh, findings.SeverityMedium} {
		for _, group := range orderedGroups {
			for _, f := range rep.Groups[group] {
				if f.Severity != sev || seen[f.ID] {
					continue
				}
				seen[f.ID] = true
				out = append(out, findingModel{ID: f.ID, Severity: f.Severity.String(), Title: f.Title, URL: f.URL, Evidence: f.Evidence[:min(1, len(f.Evidence))], Rec: f.Recommendation})
				if len(out) == 8 {
					return out
				}
			}
		}
	}
	return out
}

// severityCounts holds the per-severity tallies for the summary banner.
type severityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Info     int
}

// summaryCounts copies the global tally into a fixed-shape struct so the
// template can render every level even when it is zero.
func summaryCounts(m map[string]int) severityCounts {
	return severityCounts{
		Critical: m["critical"],
		High:     m["high"],
		Medium:   m["medium"],
		Low:      m["low"],
		Info:     m["info"],
	}
}

// skipRow is one skipped-URL reason and its count.
type skipRow struct {
	Reason string
	Count  int
}

// skippedRows converts the skipped-count map into a sorted slice for the
// template.
func skippedRows(m map[string]int) []skipRow {
	if len(m) == 0 {
		return nil
	}
	rows := make([]skipRow, 0, len(m))
	for reason, n := range m {
		rows = append(rows, skipRow{Reason: reason, Count: n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Reason < rows[j].Reason })
	return rows
}

// groupModel is one analysis group and its findings for the template.
type groupModel struct {
	Title    string
	Severity string
	Findings []findingModel
}

// findingModel is a template-safe view of a finding.
type findingModel struct {
	ID       string
	Severity string
	Title    string
	URL      string
	Evidence []string
	Rec      string
}

// reportGroups builds the ordered, non-empty groups for the template.
func reportGroups(rep *Report) []groupModel {
	var groups []groupModel
	for _, group := range orderedGroups {
		fs := rep.Groups[group]
		if len(fs) == 0 {
			continue
		}
		gm := groupModel{Title: findingsCategoryName(group), Severity: mostSevereSeverity(fs)}
		for _, f := range fs {
			gm.Findings = append(gm.Findings, findingModel{
				ID:       f.ID,
				Severity: f.Severity.String(),
				Title:    f.Title,
				URL:      f.URL,
				Evidence: f.Evidence,
				Rec:      f.Recommendation,
			})
		}
		groups = append(groups, gm)
	}
	return groups
}

// mostSevereSeverity returns the most severe finding's name in the group.
func mostSevereSeverity(fs []findings.Finding) string {
	var worst findings.Severity
	for _, f := range fs {
		if f.Severity > worst {
			worst = f.Severity
		}
	}
	return worst.String()
}
