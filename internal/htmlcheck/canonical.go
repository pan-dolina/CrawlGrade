package htmlcheck

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

// MaxCanonicals bounds the canonical declarations recorded per page.
const MaxCanonicals = 20

// Canonical is one rel=canonical declaration.
type Canonical struct {
	Href     string `json:"href"`
	URL      string `json:"url,omitempty"` // normalized absolute URL; empty if invalid
	Source   string `json:"source"`        // "html" or "header"
	Relative bool   `json:"relative,omitempty"`
	InHead   bool   `json:"in_head"`
	Error    string `json:"error,omitempty"`
}

func (p *Page) addCanonical(href, source string, inHead bool) {
	if len(p.Canonicals) >= MaxCanonicals {
		return
	}
	href = strings.TrimSpace(href)
	c := Canonical{Href: clip(href), Source: source, InHead: inHead}
	ref, err := url.Parse(href)
	switch {
	case href == "":
		c.Error = "empty href"
	case err != nil:
		c.Error = "unparsable URL"
	default:
		c.Relative = !ref.IsAbs()
		if u, err := urlnorm.Resolve(p.base, href); err != nil {
			c.Error = err.Error()
		} else {
			c.URL = u.String()
		}
	}
	p.Canonicals = append(p.Canonicals, c)
}

// AddHeaderCanonicals records canonical links from HTTP Link headers
// (RFC 8288), for example: Link: <https://example.com/a>; rel="canonical".
func (p *Page) AddHeaderCanonicals(values []string) {
	for _, v := range values {
		for _, link := range splitLinkHeader(v) {
			target, params, ok := strings.Cut(link, ";")
			target = strings.TrimSpace(target)
			if !ok || !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
				continue
			}
			for _, param := range strings.Split(params, ";") {
				k, val, _ := strings.Cut(param, "=")
				if strings.EqualFold(strings.TrimSpace(k), "rel") && hasToken(strings.Trim(strings.TrimSpace(val), `"`), "canonical") {
					p.addCanonical(target[1:len(target)-1], "header", true)
				}
			}
		}
	}
}

// splitLinkHeader splits a Link header on commas outside <...> and quotes.
func splitLinkHeader(v string) []string {
	var out []string
	depth, quoted, start := 0, false, 0
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '<':
			if !quoted {
				depth++
			}
		case '>':
			if !quoted && depth > 0 {
				depth--
			}
		case '"':
			quoted = !quoted
		case ',':
			if depth == 0 && !quoted {
				out = append(out, v[start:i])
				start = i + 1
			}
		}
	}
	return append(out, v[start:])
}

// CanonicalURL returns the effective canonical URL: the first valid
// declaration in the head or HTTP header. Empty if there is none.
func (p *Page) CanonicalURL() string {
	for _, c := range p.Canonicals {
		if c.URL != "" && c.InHead {
			return c.URL
		}
	}
	return ""
}

// distinctTargets returns the distinct valid canonical URLs.
func (p *Page) distinctTargets() []string {
	var out []string
	for _, c := range p.Canonicals {
		if c.URL != "" && !slices.Contains(out, c.URL) {
			out = append(out, c.URL)
		}
	}
	return out
}

// CanonicalFindings checks the canonical declarations of one page. scope
// identifies the audited site.
func (p *Page) CanonicalFindings(scope urlnorm.Scope) []findings.Finding {
	var out []findings.Finding
	if len(p.Canonicals) == 0 {
		return append(out, findings.CanonicalMissing.New(p.URL))
	}
	var ev, invalid, relative, body []string
	for _, c := range p.Canonicals {
		line := fmt.Sprintf("%s: %s", c.Source, c.Href)
		ev = append(ev, line)
		switch {
		case c.Error != "":
			invalid = append(invalid, line+" ("+c.Error+")")
			continue
		case c.Relative:
			relative = append(relative, line+" -> "+c.URL)
		}
		if !c.InHead {
			body = append(body, line)
		}
	}
	if len(p.distinctTargets()) > 1 || len(invalid) > 0 && len(p.Canonicals) > 1 {
		out = append(out, findings.CanonicalMultiple.New(p.URL, ev...))
	}
	if len(invalid) > 0 {
		out = append(out, findings.CanonicalInvalid.New(p.URL, invalid...))
	}
	if len(relative) > 0 {
		out = append(out, findings.CanonicalRelative.New(p.URL, relative...))
	}
	if len(body) > 0 {
		out = append(out, findings.CanonicalOutsideHead.New(p.URL, body...))
	}
	if c := p.CanonicalURL(); c != "" {
		if u, err := url.Parse(c); err == nil && !scope.Contains(u) {
			out = append(out, findings.CanonicalCrossDomain.New(p.URL, "canonical: "+c))
		}
	}
	return out
}

// Target describes what is known about a canonical target URL.
type Target struct {
	Status    int    // final HTTP status; 0 if the fetch failed
	FinalURL  string // after redirects
	Redirects int
	Canonical string // the target page's own canonical URL, if HTML
	Error     string
}

// CanonicalSiteFindings checks canonical targets using lookup, which
// returns what the crawl learned about a URL. Targets that were not fetched
// are skipped.
func CanonicalSiteFindings(pages []*Page, lookup func(string) (Target, bool)) []findings.Finding {
	var out []findings.Finding
	for _, p := range pages {
		c := p.CanonicalURL()
		if c == "" || c == p.URL {
			continue
		}
		t, ok := lookup(c)
		if !ok {
			continue
		}
		switch {
		case t.Error != "":
			out = append(out, findings.CanonicalTargetBroken.New(p.URL, "canonical: "+c, "error: "+t.Error))
			continue
		case t.Status >= 400:
			out = append(out, findings.CanonicalTargetBroken.New(p.URL, "canonical: "+c, fmt.Sprintf("HTTP status %d", t.Status)))
			continue
		case t.Redirects > 0:
			out = append(out, findings.CanonicalTargetRedirects.New(p.URL, "canonical: "+c, "redirects to "+t.FinalURL))
			continue
		}
		// Follow canonical declarations from the target to detect chains
		// and loops. The walk is bounded by the number of pages.
		chain := []string{p.URL, c}
		next := t.Canonical
		for steps := 0; next != "" && steps < len(pages)+1; steps++ {
			if next == chain[len(chain)-1] {
				break // self-canonical: end of chain
			}
			if slices.Contains(chain, next) {
				out = append(out, findings.CanonicalLoop.New(p.URL, strings.Join(append(chain, next), " -> ")))
				chain = nil
				break
			}
			chain = append(chain, next)
			nt, ok := lookup(next)
			if !ok {
				break
			}
			next = nt.Canonical
		}
		if len(chain) > 2 {
			out = append(out, findings.CanonicalChain.New(p.URL, strings.Join(chain, " -> ")))
		}
	}
	findings.Sort(out)
	return out
}
