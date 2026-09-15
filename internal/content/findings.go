package content

import (
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/terms"
)

// Findings returns the per-page content findings for the extracted content.
// It reports when the main content is mostly boilerplate, when it is too short
// to be useful, and when extraction left no tokens behind.
func (r *Result) Findings() []findings.Finding {
	var out []findings.Finding
	if r.NoText {
		out = append(out, findings.ContentNoText.New(r.URL))
		return out
	}
	if r.Boilerplate {
		out = append(out, findings.ContentBoilerplate.New(r.URL))
	}
	if tokenCount(r.Text) < MinUsefulTokens {
		out = append(out, findings.ContentVeryShort.New(r.URL))
	}
	return out
}

// MinUsefulTokens is the number of stopword-filtered tokens below which a
// page's main content is considered too short to describe a topic.
const MinUsefulTokens = 40

// tokenCount counts the meaningful tokens in s. It reuses the terms package
// so the count matches the term-strength analysis exactly.
func tokenCount(s string) int {
	return len(terms.Tokenize(s))
}
