package terms

import (
	"sort"
	"strings"
)

// PageTerms holds the n-gram profile of one crawled page.
type PageTerms struct {
	URL    string
	Counts map[string]int // n-gram string -> occurrences on the page
	Total  int            // total n-grams on the page
}

// SiteTerms aggregates n-gram profiles across the whole crawl.
type SiteTerms struct {
	// PerPage maps a page URL to its n-gram profile.
	PerPage map[string]*PageTerms
	// SiteCount maps an n-gram string to how many pages mention it.
	SiteCount map[string]int
	// TotalCount maps an n-gram string to how many times it appears on all
	// pages combined.
	TotalCount map[string]int
}

// NewSiteTerms builds a SiteTerms from the per-page n-gram profiles.
func NewSiteTerms(pages []*PageTerms) *SiteTerms {
	s := &SiteTerms{
		PerPage:    map[string]*PageTerms{},
		SiteCount:  map[string]int{},
		TotalCount: map[string]int{},
	}
	for _, p := range pages {
		if p == nil || p.URL == "" {
			continue
		}
		s.PerPage[p.URL] = p
		for gram, n := range p.Counts {
			s.TotalCount[gram] += n
			if n > 0 {
				s.SiteCount[gram]++
			}
		}
	}
	return s
}

// TopSite returns the n-grams whose site-wide strength is at least the
// threshold, sorted by descending strength then alphabetically. strength is
// the fraction of pages mentioning the n-gram, in [0,1].
func (s *SiteTerms) TopSite(minStrength float64, limit int) []NamedScore {
	type entry struct {
		gram     string
		strength float64
		count    int
	}
	var entries []entry
	for gram, pages := range s.SiteCount {
		strength := float64(pages) / float64(len(s.PerPage))
		if strength < minStrength {
			continue
		}
		entries = append(entries, entry{gram, strength, s.TotalCount[gram]})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].strength != entries[j].strength {
			return entries[i].strength > entries[j].strength
		}
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].gram < entries[j].gram
	})
	limit = min(limit, len(entries))
	out := make([]NamedScore, 0, limit)
	for _, e := range entries[:limit] {
		out = append(out, NamedScore{e.gram, e.strength, e.count})
	}
	return out
}

// NamedScore pairs an n-gram with its site-wide strength and total count.
type NamedScore struct {
	Name     string
	Strength float64 // fraction of pages mentioning the n-gram, in [0,1]
	Count    int     // total occurrences across the site
}

// String returns the n-gram and a rounded strength for display.
func (n NamedScore) String() string {
	return n.Name + " (site strength " + formatStrength(n.Strength) + ")"
}

// formatStrength renders a fraction as a percentage without importing math.
func formatStrength(s float64) string {
	percent := s * 100
	// Round to one decimal place.
	rounded := percent * 10
	rounded += 0.5
	intPart := int(rounded) / 10
	fracPart := int(rounded) % 10
	return strings.TrimSpace(itoa(intPart) + "." + itoa(fracPart) + "%")
}

// itoa converts a small non-negative integer to decimal text.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
