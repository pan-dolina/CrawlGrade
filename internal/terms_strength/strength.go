// Package terms_strength measures how strongly n-grams are associated with the
// site and with individual pages, and reports the term-strength findings.
//
// A term's site-wide strength is the fraction of indexable pages that mention
// it. A term that dominates a single page is page-specific; a term spread
// across many pages is site-wide. Pages that share the same dominant n-grams
// are reinforced as near-duplicates.
package terms_strength

import (
	"slices"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/content"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/terms"
)

// DominantTop is the number of dominant n-grams considered per page.
const DominantTop = 10

// Result reports the term-strength findings.
type Result struct {
	// SiteWide lists n-grams that appear on many pages.
	SiteWide []terms.NamedScore
	// PageSpecific maps a page URL to its dominant, page-specific n-grams.
	PageSpecific map[string][]terms.Ngram
	// DuplicateProfile lists pages that share the same dominant n-grams.
	DuplicateProfile []string
}

// Analyze measures term strength across the pages and returns the findings.
func Analyze(pages []*content.Result) *Result {
	termPages := make([]*terms.PageTerms, 0, len(pages))
	for _, r := range pages {
		termPages = append(termPages, &terms.PageTerms{
			URL:    r.URL,
			Counts: terms.Count(terms.Tokenize(r.Text)),
		})
	}
	site := terms.NewSiteTerms(termPages)

	res := &Result{
		SiteWide:     site.TopSite(0.5, 20),
		PageSpecific: map[string][]terms.Ngram{},
	}

	// Per page: dominant n-grams that are specific to that page (mentioned on
	// only that page across the site).
	seen := map[string]bool{}
	for _, p := range termPages {
		dominant := terms.Top(p.Counts, DominantTop)
		var specific []terms.Ngram
		for _, g := range dominant {
			if site.SiteCount[g.String()] <= 1 {
				specific = append(specific, g)
			}
		}
		if len(specific) > 0 {
			res.PageSpecific[p.URL] = specific
		}
	}

	// Pages that share the same dominant n-grams are reinforced duplicates.
	// Use the full dominant signature (not just page-specific n-grams) so
	// that two pages with identical content are matched even when the
	// shared n-grams are common across the site.
	groups := map[string][]string{}
	for _, p := range termPages {
		dominant := terms.Top(p.Counts, DominantTop)
		sig := make([]string, 0, len(dominant))
		for _, g := range dominant {
			sig = append(sig, g.String())
		}
		key := strings.Join(sig, "|")
		if key == "" {
			continue
		}
		groups[key] = append(groups[key], p.URL)
	}
	for _, us := range groups {
		if len(us) < 2 {
			continue
		}
		slices.Sort(us)
		for _, u := range us[1:] {
			if !seen[u] {
				res.DuplicateProfile = append(res.DuplicateProfile, u)
				seen[u] = true
			}
		}
	}
	return res
}

// Findings returns the term-strength findings.
func (r *Result) Findings() []findings.Finding {
	var out []findings.Finding
	for _, n := range r.SiteWide {
		out = append(out, findings.TermCommon.New("", n.String()))
	}
	for url, gs := range r.PageSpecific {
		var evidence []string
		for _, g := range gs {
			evidence = append(evidence, g.String())
		}
		out = append(out, findings.TermPageSpecific.New(url, evidence...))
	}
	for _, u := range r.DuplicateProfile {
		out = append(out, findings.TermDuplicate.New(u))
	}
	return out
}
