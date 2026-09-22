// Package report defines the data model shared by every CrawlGrade report
// renderer and the logic that turns a crawl and its analyses into that model.
//
// A Report is a versioned, self-describing snapshot of one audit. It is the
// single input to the terminal, JSON and HTML renderers, and to the baseline
// diff, so every format shows the same information and a saved report can be
// compared against a later one.
package report

import (
	"github.com/pan-dolina/crawlgrade/internal/terms"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// SchemaVersion identifies the shape of the Report document. It is stored in
// every report so a future reader can tell a v1 report from a v2 without
// guessing. Bump it whenever the structure changes in a backwards-incompatible
// way.
const SchemaVersion = "1"

// Summary is the headline information about a crawl, independent of the
// findings it produced.
type Summary struct {
	// StartURL is the URL the audit began from.
	StartURL string `json:"start_url"`
	// Pages is the number of pages the crawl fetched.
	Pages int `json:"pages"`
	// Findings tallies findings by severity. Every level is present so a
	// report stays stable-shaped even when a category is empty.
	Findings map[string]int `json:"findings"`
	// Categories lists the categories that produced at least one finding, in
	// report order.
	Categories []string `json:"categories"`
	// Highest is the most severe finding, or "" when there are none.
	Highest string `json:"highest_severity"`
	// StopReason explains why the crawl ended, if it did not complete.
	StopReason string `json:"stop_reason,omitempty"`
	// Skipped records how many discovered URLs were not fetched, grouped by
	// reason.
	Skipped map[string]int `json:"skipped,omitempty"`
}

// Report is one complete audit: a summary plus the findings that came from
// each analysis, grouped by source.
type Report struct {
	Baseline *Diff         `json:"baseline_diff,omitempty"`
	Pages    []PageMetric  `json:"pages,omitempty"`
	Terms    []terms.Score `json:"terms,omitempty"`
	Scores   *Scores       `json:"scores,omitempty"`
	// Version is the schema version (see SchemaVersion).
	Version string `json:"schema_version"`
	// Summary is the headline information about the crawl.
	Summary Summary `json:"summary"`
	// Findings are grouped by the analysis that produced them. Each group is
	// already sorted by the findings package.
	Groups map[string][]findings.Finding `json:"findings"`
	// Created is when the audit ran, in the local timezone. It is not part
	// of the diff, which compares findings only.
	Created time.Time `json:"created"`
}

// Group names a category of findings within a report. Group names are part of
// the public contract and must not change.
const (
	GroupCrawl      = "crawlability"
	GroupMetadata   = "metadata"
	GroupContent    = "content"
	GroupStructured = "structured-data"
	GroupLinking    = "internal-linking"
	GroupWebHygiene = "web-hygiene"
)

// New builds a Report from a crawl result and the findings each analysis
// produced. Analyses pass their findings under a group name; unknown groups
// are kept as-is so new analyses do not break existing reports.
//
// The groups are: crawl, metadata, content, structured-data, internal-linking
// and web-hygiene. The caller is responsible for having sorted each slice.
func New(start string, crawl *crawler.Result, groups map[string][]findings.Finding) *Report {
	all := make([]findings.Finding, 0, len(groups)*10)
	for _, fs := range groups {
		findings.Sort(fs)
		all = append(all, fs...)
	}
	findings.Sort(all)

	rep := &Report{
		Version: SchemaVersion,
		Created: time.Now(),
		Groups:  groups,
	}
	rep.Summary = buildSummary(start, crawl, all, groups)
	return rep
}

// buildSummary computes the headline numbers from the findings and crawl.
func buildSummary(start string, crawl *crawler.Result, all []findings.Finding, groups map[string][]findings.Finding) Summary {
	s := Summary{
		StartURL: start,
		Findings: findings.Counts(all),
		Skipped:  map[string]int{},
	}
	if crawl != nil {
		s.Pages = len(crawl.Pages)
		if crawl.StopReason != "" {
			s.StopReason = string(crawl.StopReason)
		}
		for reason, n := range crawl.SkippedCount {
			s.Skipped[reason] = n
		}
	}
	if len(all) > 0 {
		s.Highest = findings.Max(all).String()
	}
	// Record the categories that contributed, in report order.
	for _, g := range orderedGroups {
		if len(groups[g]) > 0 {
			s.Categories = append(s.Categories, g)
		}
	}
	return s
}

// orderedGroups is the fixed order categories appear in reports.
var orderedGroups = []string{
	GroupCrawl, GroupMetadata, GroupContent, GroupStructured, GroupLinking, GroupWebHygiene,
}

// FindingCount returns the number of findings in a group.
func (r *Report) FindingCount(group string) int {
	return len(r.Groups[group])
}

// Findings returns the findings of one group, or nil when the group is empty.
func (r *Report) Findings(group string) []findings.Finding {
	return r.Groups[group]
}
