package htmlcheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Heading is an h1-h6 element.
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// CheckOptions tunes heuristic checks.
type CheckOptions struct {
	// AllowMultipleH1 disables the multiple-H1 finding. HTML5 permits
	// several h1 elements and search engines handle them; some teams still
	// prefer a single h1 per page.
	AllowMultipleH1 bool
}

// headingText is the text of a heading, falling back to the alt text of
// images inside it (a logo in an h1 is a common, valid pattern).
func headingText(n *html.Node) string {
	if t := text(n); t != "" {
		return t
	}
	var alts []string
	walk(n, func(c *html.Node) bool {
		if c.Type == html.ElementNode && c.DataAtom == atom.Img {
			if a := collapse(attr(c, "alt")); a != "" {
				alts = append(alts, a)
			}
		}
		return true
	})
	return strings.Join(alts, " ")
}

// H1s returns the text of all h1 headings.
func (p *Page) H1s() []string {
	var out []string
	for _, h := range p.Headings {
		if h.Level == 1 {
			out = append(out, h.Text)
		}
	}
	return out
}

// HeadingFindings checks the heading outline of one page.
func (p *Page) HeadingFindings(opts CheckOptions) []findings.Finding {
	var out []findings.Finding
	h1 := p.H1s()
	switch {
	case len(h1) == 0:
		out = append(out, findings.HeadingNoH1.New(p.URL))
	case len(h1) > 1 && !opts.AllowMultipleH1:
		out = append(out, findings.HeadingMultipleH1.New(p.URL, quoteAll(h1)...))
	}

	var empty, skips []string
	prev := 0
	for i, h := range p.Headings {
		if h.Text == "" {
			empty = append(empty, fmt.Sprintf("heading %d: empty <h%d>", i+1, h.Level))
		}
		switch {
		case prev == 0 && h.Level > 1:
			skips = append(skips, fmt.Sprintf("document starts with <h%d> %q", h.Level, h.Text))
		case prev > 0 && h.Level > prev+1:
			skips = append(skips, fmt.Sprintf("<h%d> follows <h%d>: %q", h.Level, prev, h.Text))
		}
		prev = h.Level
	}
	if len(empty) > 0 {
		out = append(out, findings.HeadingEmpty.New(p.URL, empty...))
	}
	if len(skips) > 0 {
		out = append(out, findings.HeadingHierarchy.New(p.URL, skips...))
	}
	return out
}

// DuplicateH1Findings reports h1 texts used on more than one page.
func DuplicateH1Findings(pages []*Page) []findings.Finding {
	return duplicates(pages, func(p *Page) string {
		h1 := p.H1s()
		if len(h1) == 0 {
			return ""
		}
		return strings.ToLower(h1[0])
	}, findings.HeadingDuplicateH1)
}

// hasToken reports whether a space-separated attribute value contains tok,
// case-insensitively (rel and similar attributes).
func hasToken(list, tok string) bool {
	return slices.ContainsFunc(strings.Fields(list), func(s string) bool { return strings.EqualFold(s, tok) })
}
