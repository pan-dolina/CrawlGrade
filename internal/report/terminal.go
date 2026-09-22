package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// severitySymbol maps a severity to a two-character marker for the terminal
// renderer. The markers are ASCII so the output is readable everywhere.
var severitySymbol = map[findings.Severity]string{
	findings.SeverityCritical: "!!",
	findings.SeverityHigh:     "! ",
	findings.SeverityMedium:   "* ",
	findings.SeverityLow:      "- ",
	findings.SeverityInfo:     ". ",
}

// RenderTerminal writes a human-readable summary of the report to w. It shows
// the headline counts, then each finding grouped by analysis, then the crawl
// notes. It never emits ANSI colour codes itself; the caller may strip or add
// them.
func RenderTerminal(w io.Writer, rep *Report) error {
	w = terminalWriter{w}
	if err := renderSummary(w, rep); err != nil {
		return err
	}
	if rep.Scores != nil {
		if _, err := fmt.Fprintf(w, "Technical SEO: %d/100; passive web hygiene: %d/100 (not rankings)\n", rep.Scores.SEO, rep.Scores.WebHygiene); err != nil {
			return err
		}
	}
	for _, term := range rep.Terms {
		if _, err := fmt.Fprintf(w, "Term: %s %.1f\n", term.Term, term.Strength); err != nil {
			return err
		}
	}
	for _, p := range rep.Pages {
		if _, err := fmt.Fprintf(w, "Page: %s status=%d inbound=%d outbound=%d\n", p.URL, p.Status, p.Inbound, p.Outbound); err != nil {
			return err
		}
	}
	if err := renderFindings(w, rep); err != nil {
		return err
	}
	if rep.Baseline != nil {
		var b strings.Builder
		rep.Baseline.RenderTerminal(&b)
		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	return renderNotes(w, rep)
}

// renderSummary writes the headline counts and the stop reason.
func renderSummary(w io.Writer, rep *Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "CrawlGrade report — %s\n", rep.Summary.StartURL)
	fmt.Fprintf(&b, "Pages crawled: %d\n", rep.Summary.Pages)
	fmt.Fprintf(&b, "Findings: %d critical, %d high, %d medium, %d low, %d info\n",
		rep.Summary.Findings["critical"], rep.Summary.Findings["high"],
		rep.Summary.Findings["medium"], rep.Summary.Findings["low"],
		rep.Summary.Findings["info"])
	if rep.Summary.StopReason != "" {
		fmt.Fprintf(&b, "Crawl stopped: %s\n", rep.Summary.StopReason)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// renderFindings writes each finding under its analysis group.
func renderFindings(w io.Writer, rep *Report) error {
	var b strings.Builder
	any := false
	for _, group := range orderedGroups {
		fs := rep.Groups[group]
		if len(fs) == 0 {
			continue
		}
		any = true
		fmt.Fprintf(&b, "\n%s\n", findings.Category(group).Name())
		for _, f := range fs {
			writeFinding(&b, f)
		}
	}
	if !any {
		b.WriteString("\nNo findings.\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeFinding renders one finding with its severity marker, title, URL and
// evidence lines.
func writeFinding(b *strings.Builder, f findings.Finding) {
	sym, ok := severitySymbol[f.Severity]
	if !ok {
		sym = "? "
	}
	fmt.Fprintf(b, "  %s [%s] %s %s\n", sym, f.Severity, f.ID, f.Title)
	if f.URL != "" {
		fmt.Fprintf(b, "      URL: %s\n", f.URL)
	}
	for _, line := range f.Evidence {
		fmt.Fprintf(b, "      - %s\n", line)
	}
	if f.Recommendation != "" {
		fmt.Fprintf(b, "      %s\n", f.Recommendation)
	}
}

// renderNotes writes crawl-level notes that are not findings, such as
// skipped URLs grouped by reason.
func renderNotes(w io.Writer, rep *Report) error {
	if len(rep.Summary.Skipped) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("\nSkipped URLs\n")
	reasons := make([]string, 0, len(rep.Summary.Skipped))
	for r := range rep.Summary.Skipped {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		fmt.Fprintf(&b, "  %s: %d\n", r, rep.Summary.Skipped[r])
	}
	_, err := io.WriteString(w, b.String())
	return err
}
