package urlnorm

import (
	"net/url"
	"sort"
	"strings"
)

// Scope decides whether a URL belongs to the audited site.
type Scope struct {
	key string
}

// NewScope returns the scope of the site start belongs to.
func NewScope(start *url.URL) Scope {
	return Scope{key: SiteKey(start)}
}

// SiteKey identifies a site: the host without a leading "www." plus any
// non-default port. The scheme is ignored so that http and https versions of
// the same site are in scope (and HTTP to HTTPS redirects stay internal).
func SiteKey(u *url.URL) string {
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	port := u.Port()
	switch {
	case port == "", u.Scheme == "http" && port == "80", u.Scheme == "https" && port == "443":
		return host
	}
	return host + ":" + port
}

// Contains reports whether u is an http(s) URL on the site.
func (s Scope) Contains(u *url.URL) bool {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	return SiteKey(u) == s.key
}

// Key returns the site key.
func (s Scope) Key() string { return s.key }

// TrapReason explains why a URL was not admitted to the crawl.
type TrapReason string

// Trap reasons.
const (
	TrapNone             TrapReason = ""
	TrapURLTooLong       TrapReason = "url-too-long"
	TrapTooManySegments  TrapReason = "too-many-path-segments"
	TrapRepeatedSegment  TrapReason = "repeated-path-segment"
	TrapQueryExplosion   TrapReason = "query-explosion"
	TrapPatternExplosion TrapReason = "path-pattern-explosion"
)

// TrapLimits configures TrapGuard.
type TrapLimits struct {
	MaxURLLength       int
	MaxPathSegments    int
	MaxSegmentRepeats  int
	MaxQueryVariants   int // distinct query strings per path
	MaxPatternVariants int // URLs per path pattern (digit runs collapsed)
}

// DefaultTrapLimits returns the documented defaults.
func DefaultTrapLimits() TrapLimits {
	return TrapLimits{
		MaxURLLength:       MaxURLLength,
		MaxPathSegments:    30,
		MaxSegmentRepeats:  3,
		MaxQueryVariants:   25,
		MaxPatternVariants: 100,
	}
}

// TrapGuard admits URLs to the crawl and rejects those that look like crawl
// traps. It is not safe for concurrent use; the crawler admits URLs from a
// single goroutine in sorted order, which keeps decisions deterministic.
// Memory grows only with admitted URLs.
type TrapGuard struct {
	limits   TrapLimits
	queries  map[string]int // path -> admitted distinct queries
	patterns map[string]int // pattern -> admitted URLs
	seen     map[string]bool
}

// NewTrapGuard returns a guard with the given limits.
func NewTrapGuard(l TrapLimits) *TrapGuard {
	d := DefaultTrapLimits()
	if l.MaxURLLength <= 0 {
		l.MaxURLLength = d.MaxURLLength
	}
	if l.MaxPathSegments <= 0 {
		l.MaxPathSegments = d.MaxPathSegments
	}
	if l.MaxSegmentRepeats <= 0 {
		l.MaxSegmentRepeats = d.MaxSegmentRepeats
	}
	if l.MaxQueryVariants <= 0 {
		l.MaxQueryVariants = d.MaxQueryVariants
	}
	if l.MaxPatternVariants <= 0 {
		l.MaxPatternVariants = d.MaxPatternVariants
	}
	return &TrapGuard{limits: l, queries: map[string]int{}, patterns: map[string]int{}, seen: map[string]bool{}}
}

// Admit decides whether a normalized URL may be crawled. Admitting the same
// URL twice is allowed and does not count twice.
func (g *TrapGuard) Admit(u *url.URL) TrapReason {
	s := u.String()
	if g.seen[s] {
		return TrapNone
	}
	if len(s) > g.limits.MaxURLLength {
		return TrapURLTooLong
	}
	segs := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(segs) > g.limits.MaxPathSegments {
		return TrapTooManySegments
	}
	counts := map[string]int{}
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		counts[seg]++
		if counts[seg] > g.limits.MaxSegmentRepeats {
			return TrapRepeatedSegment
		}
	}
	path := u.EscapedPath()
	if u.RawQuery != "" && g.queries[path] >= g.limits.MaxQueryVariants {
		return TrapQueryExplosion
	}
	pattern := Pattern(u)
	if g.patterns[pattern] >= g.limits.MaxPatternVariants {
		return TrapPatternExplosion
	}
	g.seen[s] = true
	if u.RawQuery != "" {
		g.queries[path]++
	}
	g.patterns[pattern]++
	return TrapNone
}

// Pattern reduces a URL to a template: runs of digits in the path become
// "{n}" and the query is reduced to its sorted parameter names. URLs of
// generated calendars and paginations share a pattern.
func Pattern(u *url.URL) string {
	var b strings.Builder
	b.WriteString(collapseDigits(u.EscapedPath()))
	if u.RawQuery != "" {
		var keys []string
		for _, part := range strings.Split(u.RawQuery, "&") {
			k, _, _ := strings.Cut(part, "=")
			keys = append(keys, collapseDigits(k))
		}
		sort.Strings(keys)
		b.WriteString("?")
		b.WriteString(strings.Join(keys, "&"))
	}
	return b.String()
}

func collapseDigits(s string) string {
	var b strings.Builder
	inDigits := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			if !inDigits {
				b.WriteString("{n}")
				inDigits = true
			}
			continue
		}
		inDigits = false
		b.WriteByte(c)
	}
	return b.String()
}
