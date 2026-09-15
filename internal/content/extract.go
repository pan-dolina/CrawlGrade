// Package content extracts the main text of a page and reports problems with
// that extraction: pages that are mostly boilerplate, pages whose main text
// is too short to be useful, and pages with no extractable text at all.
//
// Extraction is heuristic and conservative. It walks the parsed document,
// treats a small set of elements as shared chrome (navigation, header,
// footer, scripts, styles) and collects the text of everything else as
// candidate main content. The result is bounded: the number of elements
// visited, the length of the collected text and the number of findings are
// all capped, so a hostile document cannot make extraction unbounded.
package content

import (
	"bytes"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/terms"
	"golang.org/x/net/html"
)

// Limits on what is extracted from one document.
const (
	// MaxContentRunes bounds the collected main text.
	MaxContentRunes = 64 << 10
	// MaxElements bounds how many elements are visited.
	MaxElements = 20000
)

// chrome lists the elements that are treated as shared site chrome and are
// excluded from main content. They are the elements that repeat across pages
// and carry no topic of their own.
var chrome = map[string]bool{
	"head": true, "script": true, "style": true, "noscript": true,
	"nav": true, "header": true, "footer": true, "aside": true,
	"form": true, "template": true, "svg": true, "math": true,
}

// Result is the extracted main content of one page.
type Result struct {
	URL         string
	Text        string // collapsed main text
	Tokens      int    // tokens after stopword removal (see terms.Tokenize)
	Boilerplate bool   // main text is a small fraction of the page
	NoText      bool   // no tokens survived extraction
}

// Extract parses body and returns its extracted main content. body is served
// for pageURL; the scope is unused here but kept for signature symmetry with
// the rest of the pipeline.
func Extract(pageURL, body string) *Result {
	r := &Result{URL: pageURL}
	root, err := html.Parse(bytes.NewReader([]byte(body)))
	if err != nil {
		return r
	}
	var b strings.Builder
	elements := 0
	walk(root, func(n *html.Node) bool {
		if elements >= MaxElements {
			return true
		}
		elements++
		if n.Type == html.ElementNode {
			data := n.Data
			if chrome[data] {
				// Skip the subtree: it is shared chrome.
				return true
			}
		}
		return false
	}, func(n *html.Node) {
		if n.Type != html.TextNode || b.Len() >= MaxContentRunes {
			return
		}
		text := n.Data
		if strings.TrimSpace(text) == "" {
			return
		}
		if len(r.Text) < MaxContentRunes {
			r.Text += " " + text
		}
	})
	r.Text = collapseSpaces(r.Text)
	if len(r.Text) > MaxContentRunes {
		r.Text = r.Text[:MaxContentRunes]
	}
	// Count meaningful tokens; a page with none is reported separately.
	r.Tokens = len(terms.Tokenize(r.Text))
	r.NoText = r.Tokens == 0
	// Boilerplate is flagged when the extracted text is a small fraction of
	// the raw text: most of the page is shared chrome.
	if rawLen := len(collapseSpaces(body)); rawLen > 0 {
		ratio := float64(len(r.Text)) / float64(rawLen)
		r.Boilerplate = ratio < BoilerplateRatio
	}
	return r
}

// BoilerplateRatio is the text-length fraction below which a page is treated
// as mostly boilerplate.
const BoilerplateRatio = 0.05

// collapseSpaces joins runs of whitespace into single spaces.
func collapseSpaces(s string) string {
	var b strings.Builder
	var lastSpace bool
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

// walk is a non-recursive depth-first traversal that visits nodes and skips
// subtrees whose visit function returns false. It bounds the number of nodes
// visited so a hostile document cannot exhaust the stack or run unbounded.
func walk(root *html.Node, skip func(*html.Node) bool, visit func(*html.Node)) {
	if root == nil {
		return
	}
	stack := []*html.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		visit(n)
		if skip != nil && skip(n) {
			continue
		}
		// Push children in reverse so they are visited in document order.
		for c := n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}
}
