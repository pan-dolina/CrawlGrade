// Package htmlcheck parses HTML documents into a page model and runs the
// per-page and site-wide metadata checks.
//
// Documents are parsed with golang.org/x/net/html, the HTML5 tree
// construction algorithm browsers use, so malformed markup is interpreted the
// way a browser would interpret it. No regular expressions are applied to
// markup.
package htmlcheck

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"

	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

// Limits on what is extracted from one document.
const (
	MaxTextRunes = 1024 // title, description, heading and anchor text
)

// Page is the parsed model of one HTML document.
type Page struct {
	URL          string      `json:"url"`
	Titles       []string    `json:"titles,omitempty"`
	Descriptions []string    `json:"descriptions,omitempty"`
	Headings     []Heading   `json:"headings,omitempty"`
	Canonicals   []Canonical `json:"canonicals,omitempty"`
	Robots       Robots      `json:"robots"`
	// Links, Images, Hreflangs and Social are populated by the walk below;
	// they are not part of the JSON model that feeds the report.
	Links     []Link     `json:"-"`
	Images    []Image    `json:"-"`
	Hreflangs []Hreflang `json:"-"`
	Social    []Social   `json:"-"`

	// Lang is the lang attribute of the <html> element; HasLang reports
	// whether the attribute was present.
	Lang    string `json:"lang,omitempty"`
	HasLang bool   `json:"-"`

	base *url.URL
	// scope identifies the audited site; links outside it are external.
	scope urlnorm.Scope
}

// MaxHeadings bounds the headings recorded per page.
const MaxHeadings = 500

// Title returns the first title, or "" if there is none.
func (p *Page) Title() string {
	if len(p.Titles) == 0 {
		return ""
	}
	return p.Titles[0]
}

// Description returns the first meta description, or "".
func (p *Page) Description() string {
	if len(p.Descriptions) == 0 {
		return ""
	}
	return p.Descriptions[0]
}

// Parse parses body, served for pageURL with the given Content-Type header
// value, and returns the page model and the document tree. The tree is
// needed for content extraction and can be discarded afterwards. scope
// identifies the audited site so that links can be classified as internal or
// external.
func Parse(body []byte, pageURL *url.URL, contentType string, scope urlnorm.Scope) (*Page, *html.Node, error) {
	var r io.Reader = bytes.NewReader(body)
	// Decode legacy encodings (for example ISO-8859-2) to UTF-8 using the
	// Content-Type header, a BOM or <meta charset>, as browsers do.
	if cr, err := charset.NewReader(r, contentType); err == nil {
		r = cr
	}
	root, err := html.Parse(r)
	if err != nil {
		return nil, nil, err
	}
	p := &Page{URL: pageURL.String(), base: pageURL, scope: scope}
	p.setBase(root)
	p.setLang(root)
	walk(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Namespace != "" {
			// SVG and MathML have their own <title> and <a> elements that do
			// not describe the document.
			return n.Type != html.ElementNode
		}
		switch n.DataAtom {
		case atom.Title:
			if inHead(n) || len(p.Titles) == 0 {
				p.Titles = append(p.Titles, clip(text(n)))
			}
		case atom.Meta:
			name := strings.ToLower(strings.TrimSpace(attr(n, "name")))
			switch {
			case name == "description":
				p.Descriptions = append(p.Descriptions, clip(collapse(attr(n, "content"))))
			default:
				if agent, ok := robotsAgent(name); ok {
					p.addRobots("meta", agent, attr(n, "content"))
				} else {
					p.addSocial(n)
				}
			}
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			if len(p.Headings) < MaxHeadings {
				p.Headings = append(p.Headings, Heading{Level: int(n.Data[1] - '0'), Text: clip(headingText(n))})
			}
			return false
		case atom.Link:
			switch {
			case hasToken(attr(n, "rel"), "canonical"):
				p.addCanonical(attr(n, "href"), "html", inHead(n))
			case hasToken(attr(n, "rel"), "alternate") && len(p.Hreflangs) < 100:
				if _, present := attrOK(n, "hreflang"); !present {
					break
				}
				if href := strings.TrimSpace(attr(n, "href")); href != "" {
					if u, err := p.Resolve(href); err == nil {
						p.Hreflangs = append(p.Hreflangs, Hreflang{Href: u.String(), Target: strings.TrimSpace(attr(n, "hreflang"))})
					}
				}
			}
		case atom.A:
			p.addLink(n)
		case atom.Img:
			p.addImage(n)
		case atom.Template:
			return false
		}
		return true
	})
	return p, root, nil
}

// setBase applies the first <base href>, as browsers do.
func (p *Page) setBase(root *html.Node) {
	var found bool
	walk(root, func(n *html.Node) bool {
		if found {
			return false
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Base && n.Namespace == "" {
			if href, ok := attrOK(n, "href"); ok {
				found = true
				if u, err := p.base.Parse(strings.TrimSpace(href)); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
					p.base = u
				}
			}
		}
		return true
	})
}

// Resolve resolves a reference found in the document against its base URL.
func (p *Page) Resolve(ref string) (*url.URL, error) {
	return urlnorm.Resolve(p.base, ref)
}

// walk visits nodes depth-first without recursion, so deeply nested
// documents cannot exhaust the goroutine stack. visit returns false to skip
// a node's children.
func walk(root *html.Node, visit func(*html.Node) bool) {
	stack := []*html.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !visit(n) {
			continue
		}
		for c := n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}
}

func inHead(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.DataAtom == atom.Head {
			return true
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	v, _ := attrOK(n, key)
	return v
}

func attrOK(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// text returns the whitespace-collapsed text content of n.
func text(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			if b.Len() < 4*MaxTextRunes {
				b.WriteString(c.Data)
				b.WriteByte(' ')
			}
		}
		if c.Type == html.ElementNode && (c.DataAtom == atom.Script || c.DataAtom == atom.Style || c.DataAtom == atom.Template) {
			return false
		}
		return true
	})
	return collapse(b.String())
}

// collapse trims and collapses runs of Unicode whitespace to single spaces.
func collapse(s string) string {
	return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ")
}

// clip limits s to MaxTextRunes runes.
func clip(s string) string {
	n := 0
	for i := range s {
		if n == MaxTextRunes {
			return s[:i]
		}
		n++
	}
	return s
}
