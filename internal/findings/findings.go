// Package findings defines the finding model shared by all analyses.
//
// A Finding is a single observation about a page or the whole site. Findings
// are instantiated from Rules, which live in a central catalog so that
// finding IDs remain stable between releases.
package findings

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Severity ranks findings. The zero value is invalid.
type Severity int

// Severities in ascending order.
const (
	SeverityInfo Severity = iota + 1
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

var severityNames = [...]string{
	SeverityInfo:     "info",
	SeverityLow:      "low",
	SeverityMedium:   "medium",
	SeverityHigh:     "high",
	SeverityCritical: "critical",
}

// Severities returns all severities from most to least severe.
func Severities() []Severity {
	return []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
}

func (s Severity) String() string {
	if s.Valid() {
		return severityNames[s]
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

// Valid reports whether s is a defined severity.
func (s Severity) Valid() bool {
	return s >= SeverityInfo && s <= SeverityCritical
}

// ParseSeverity parses a severity name case-insensitively.
func ParseSeverity(name string) (Severity, error) {
	for _, s := range Severities() {
		if strings.EqualFold(severityNames[s], name) {
			return s, nil
		}
	}
	return 0, fmt.Errorf("unknown severity %q", name)
}

// MarshalText implements encoding.TextMarshaler.
func (s Severity) MarshalText() ([]byte, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("invalid severity %d", int(s))
	}
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Severity) UnmarshalText(b []byte) error {
	v, err := ParseSeverity(string(b))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// Category groups findings in reports and scores.
type Category string

// Categories, in report order.
const (
	CategoryCrawlability    Category = "crawlability"
	CategoryMetadata        Category = "metadata"
	CategoryContent         Category = "content"
	CategoryStructuredData  Category = "structured-data"
	CategoryInternalLinking Category = "internal-linking"
	CategoryWebHygiene      Category = "web-hygiene"
)

// Categories returns all categories in report order.
func Categories() []Category {
	return []Category{
		CategoryCrawlability, CategoryMetadata, CategoryContent,
		CategoryStructuredData, CategoryInternalLinking, CategoryWebHygiene,
	}
}

// Name returns the human-readable category name.
func (c Category) Name() string {
	switch c {
	case CategoryCrawlability:
		return "Crawlability"
	case CategoryMetadata:
		return "Metadata"
	case CategoryContent:
		return "Content"
	case CategoryStructuredData:
		return "Structured Data"
	case CategoryInternalLinking:
		return "Internal Linking"
	case CategoryWebHygiene:
		return "Web Hygiene"
	}
	return string(c)
}

// Valid reports whether c is a defined category.
func (c Category) Valid() bool {
	return slices.Contains(Categories(), c)
}

// Rule is the static definition of a finding type.
type Rule struct {
	ID             string
	Category       Category
	Severity       Severity
	Title          string
	Description    string
	Recommendation string
}

// Finding is a concrete observation.
type Finding struct {
	ID             string   `json:"id"`
	Severity       Severity `json:"severity"`
	Category       Category `json:"category"`
	URL            string   `json:"url,omitempty"`
	Title          string   `json:"title"`
	Evidence       []string `json:"evidence,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
}

// Evidence limits keep reports readable and bounded when a site repeats a
// problem thousands of times.
const (
	MaxEvidenceLines = 20
	MaxEvidenceRunes = 300
)

// New instantiates a finding for url. url may be empty for site-wide
// findings. Evidence is truncated to MaxEvidenceLines lines of at most
// MaxEvidenceRunes runes each.
func (r Rule) New(url string, evidence ...string) Finding {
	return Finding{
		ID:             r.ID,
		Severity:       r.Severity,
		Category:       r.Category,
		URL:            url,
		Title:          r.Title,
		Evidence:       LimitEvidence(evidence),
		Recommendation: r.Recommendation,
	}
}

// WithSeverity returns a copy of f with a different severity. Rules define a
// default; context can make the same condition more or less severe.
func (f Finding) WithSeverity(s Severity) Finding {
	f.Severity = s
	return f
}

// LimitEvidence applies the evidence limits to lines.
func LimitEvidence(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	n := min(len(lines), MaxEvidenceLines)
	out := make([]string, 0, n+1)
	for _, l := range lines[:n] {
		out = append(out, Truncate(l, MaxEvidenceRunes))
	}
	if extra := len(lines) - n; extra > 0 {
		out = append(out, fmt.Sprintf("... and %d more", extra))
	}
	return out
}

// Truncate shortens s to at most maxRunes runes, marking truncation with an
// ellipsis. Invalid UTF-8 is replaced.
func Truncate(s string, maxRunes int) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return "…"
	}
	i, n := 0, 0
	for i = range s {
		if n == maxRunes-1 {
			break
		}
		n++
	}
	return s[:i] + "…"
}

// Sort orders findings deterministically: most severe first, then by ID,
// URL and evidence.
func Sort(fs []Finding) {
	slices.SortStableFunc(fs, Compare)
}

// Compare implements the ordering used by Sort.
func Compare(a, b Finding) int {
	return cmp.Or(
		cmp.Compare(b.Severity, a.Severity),
		cmp.Compare(a.ID, b.ID),
		cmp.Compare(a.URL, b.URL),
		slices.Compare(a.Evidence, b.Evidence),
	)
}

// Max returns the highest severity in fs, or 0 if fs is empty.
func Max(fs []Finding) Severity {
	var m Severity
	for _, f := range fs {
		m = max(m, f.Severity)
	}
	return m
}

// Counts tallies findings by severity name. All severities are present.
func Counts(fs []Finding) map[string]int {
	c := make(map[string]int, len(severityNames))
	for _, s := range Severities() {
		c[s.String()] = 0
	}
	for _, f := range fs {
		if f.Severity.Valid() {
			c[f.Severity.String()]++
		}
	}
	return c
}

// AtLeast returns the findings whose severity is >= threshold.
func AtLeast(fs []Finding, threshold Severity) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Severity >= threshold {
			out = append(out, f)
		}
	}
	return out
}
