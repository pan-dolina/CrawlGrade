// Package robots parses robots.txt files and matches URLs against them
// following RFC 9309.
//
// Parsing never fails: unknown or malformed lines are skipped and recorded
// as warnings, as crawlers do. Every allocation is bounded by the input size
// (itself limited when fetching) and by explicit rule and pattern limits.
package robots

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits applied while parsing.
const (
	// MaxParseBytes is the prefix of the file that is parsed. RFC 9309
	// requires crawlers to parse at least 500 KiB.
	MaxParseBytes  = 500 << 10
	MaxRules       = 10000
	MaxPatternLen  = 2048
	MaxSitemaps    = 500
	maxWarnings    = 50
	maxLineLength  = 8192
	ProductToken   = "crawlgrade"
	robotsTxtPath  = "/robots.txt"
	maxAgentLength = 256
)

// Rule is one allow or disallow line.
type Rule struct {
	Allow   bool
	Pattern string // percent-encoding normalized
	Line    int
}

type group struct {
	agents []string
	rules  []Rule
	delay  time.Duration
	hasDly bool
}

// Warning is a line that was skipped.
type Warning struct {
	Line    int
	Message string
}

// Robots is a parsed robots.txt file.
type Robots struct {
	groups    []group
	Sitemaps  []SitemapRef
	Warnings  []Warning
	Truncated bool // the file was longer than MaxParseBytes or MaxRules
	RuleCount int
}

// SitemapRef is a Sitemap directive.
type SitemapRef struct {
	URL  string
	Line int
}

// Parse parses a robots.txt body.
func Parse(data []byte) *Robots {
	r := &Robots{}
	if len(data) > MaxParseBytes {
		data = data[:MaxParseBytes]
		r.Truncated = true
	}
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))

	var cur *group
	lastWasAgent := false
	lineNo := 0
	for len(data) > 0 {
		lineNo++
		var line []byte
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			line = data[:i]
			if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
				i++
			}
			data = data[i+1:]
		} else {
			line, data = data, nil
		}
		if len(line) > maxLineLength {
			r.warn(lineNo, "line too long")
			continue
		}
		if i := bytes.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		text := strings.TrimSpace(strings.ToValidUTF8(string(line), "\uFFFD"))
		if text == "" {
			continue
		}
		key, value, ok := strings.Cut(text, ":")
		if !ok {
			r.warn(lineNo, "line has no ':' separator")
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch key {
		case "user-agent", "useragent", "user agent":
			if !lastWasAgent || cur == nil {
				r.groups = append(r.groups, group{})
				cur = &r.groups[len(r.groups)-1]
			}
			agent := strings.ToLower(value)
			if len(agent) > maxAgentLength {
				agent = agent[:maxAgentLength]
			}
			cur.agents = append(cur.agents, agent)
			lastWasAgent = true
			continue
		case "allow", "disallow":
			lastWasAgent = false
			if cur == nil {
				r.warn(lineNo, key+" outside of a user-agent group")
				continue
			}
			if value == "" {
				// An empty Disallow allows everything; it adds no rule.
				continue
			}
			if len(value) > MaxPatternLen {
				r.warn(lineNo, "pattern too long")
				continue
			}
			if r.RuleCount >= MaxRules {
				r.Truncated = true
				continue
			}
			r.RuleCount++
			cur.rules = append(cur.rules, Rule{Allow: key == "allow", Pattern: normalizePattern(value), Line: lineNo})
		case "sitemap":
			// Sitemap lines are global and do not end a user-agent run.
			if len(r.Sitemaps) < MaxSitemaps && value != "" {
				r.Sitemaps = append(r.Sitemaps, SitemapRef{URL: value, Line: lineNo})
			}
		case "crawl-delay":
			lastWasAgent = false
			if cur == nil {
				r.warn(lineNo, "crawl-delay outside of a user-agent group")
				continue
			}
			secs, err := strconv.ParseFloat(value, 64)
			if err != nil || secs < 0 || secs > 86400 {
				r.warn(lineNo, "invalid crawl-delay")
				continue
			}
			cur.delay, cur.hasDly = time.Duration(secs*float64(time.Second)), true
		default:
			// Unknown keys (host, clean-param, noindex, ...) are ignored by
			// RFC 9309 crawlers. They do not end the user-agent line run.
			r.warn(lineNo, fmt.Sprintf("unknown directive %q", truncate(key, 40)))
			continue
		}
	}
	return r
}

func (r *Robots) warn(line int, msg string) {
	if len(r.Warnings) < maxWarnings {
		r.Warnings = append(r.Warnings, Warning{Line: line, Message: msg})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// agentToken reduces a user-agent line value to its product token.
func agentToken(v string) string {
	if i := strings.IndexAny(v, "/ \t"); i >= 0 {
		v = v[:i]
	}
	return v
}

// rulesFor returns the rules of all groups matching agent (merged, as RFC
// 9309 requires), falling back to the "*" groups.
func (r *Robots) rulesFor(agent string) (rules []Rule, delay time.Duration, hasDelay, specific bool) {
	agent = strings.ToLower(agent)
	collect := func(match func(string) bool) bool {
		found := false
		for _, g := range r.groups {
			for _, a := range g.agents {
				if match(a) {
					rules = append(rules, g.rules...)
					if g.hasDly && !hasDelay {
						delay, hasDelay = g.delay, true
					}
					found = true
					break
				}
			}
		}
		return found
	}
	if agent != "*" && collect(func(a string) bool { return a != "*" && agentToken(a) == agent }) {
		return rules, delay, hasDelay, true
	}
	collect(func(a string) bool { return a == "*" })
	return rules, delay, hasDelay, false
}

// HasGroupFor reports whether a group names agent explicitly.
func (r *Robots) HasGroupFor(agent string) bool {
	_, _, _, specific := r.rulesFor(agent)
	return specific
}

// CrawlDelay returns the Crawl-delay for agent, if any.
func (r *Robots) CrawlDelay(agent string) (time.Duration, bool) {
	_, d, ok, _ := r.rulesFor(agent)
	return d, ok
}

// Match returns whether pathQuery (an escaped path with optional query) is
// allowed for agent, and the deciding rule (nil when no rule matched).
func (r *Robots) Match(agent, pathQuery string) (bool, *Rule) {
	if pathQuery == "" {
		pathQuery = "/"
	}
	if pathQuery == robotsTxtPath {
		return true, nil
	}
	target := normalizePattern(pathQuery)
	rules, _, _, _ := r.rulesFor(agent)
	var best *Rule
	bestLen := -1
	for i := range rules {
		rule := &rules[i]
		if !match(rule.Pattern, target) {
			continue
		}
		l := len(rule.Pattern)
		if l > bestLen || (l == bestLen && rule.Allow && !best.Allow) {
			best, bestLen = rule, l
		}
	}
	if best == nil {
		return true, nil
	}
	return best.Allow, best
}

// Allowed reports whether pathQuery is allowed for agent.
func (r *Robots) Allowed(agent, pathQuery string) bool {
	ok, _ := r.Match(agent, pathQuery)
	return ok
}

// match reports whether pattern matches the start of target. "*" matches
// any sequence and a trailing "$" anchors the end. The algorithm is the
// linear-backtracking glob match: O(len(pattern) * len(target)) worst case,
// with no recursion.
func match(pattern, target string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	if anchored {
		pattern = pattern[:len(pattern)-1]
	}
	p, t := 0, 0
	star, mark := -1, 0
	for {
		if p == len(pattern) {
			if !anchored || t == len(target) {
				return true
			}
			// Anchored pattern must consume the whole target: backtrack.
			if star < 0 {
				return false
			}
			mark++
			if mark > len(target) {
				return false
			}
			p, t = star+1, mark
			continue
		}
		if pattern[p] == '*' {
			star, mark = p, t
			p++
			continue
		}
		if t < len(target) && pattern[p] == target[t] {
			p++
			t++
			continue
		}
		if star < 0 || mark >= len(target) {
			return false
		}
		mark++
		p, t = star+1, mark
	}
}

const upperHex = "0123456789ABCDEF"

// normalizePattern percent-encodes non-ASCII and control bytes and upper-
// cases existing escapes, and decodes escaped unreserved characters, so that
// patterns and URL paths compare byte for byte.
func normalizePattern(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			v := unhex(s[i+1])<<4 | unhex(s[i+2])
			if isUnreserved(v) {
				b.WriteByte(v)
			} else {
				b.WriteByte('%')
				b.WriteByte(upperHex[v>>4])
				b.WriteByte(upperHex[v&15])
			}
			i += 2
			continue
		}
		if c <= 0x20 || c >= 0x7f {
			b.WriteByte('%')
			b.WriteByte(upperHex[c>>4])
			b.WriteByte(upperHex[c&15])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return c - 'A' + 10
}

func isUnreserved(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
		c == '-' || c == '.' || c == '_' || c == '~'
}
