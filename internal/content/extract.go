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
	root, err := html.Parse(bytes.NewReader([]byte(body)))
	if err != nil {
		return &Result{URL: pageURL, NoText: true}
	}
	return ExtractTree(pageURL, root)
}

// ExtractTree uses the already decoded HTML tree. Explicit main/article regions
// take precedence; navigation and hidden elements are excluded.
func ExtractTree(pageURL string, root *html.Node) *Result {
	r := &Result{URL: pageURL}
	var main *html.Node
	walk(root, func(n *html.Node) bool { return n.Type == html.ElementNode && chrome[n.Data] }, func(n *html.Node) {
		if main == nil && n.Type == html.ElementNode && (n.Data == "main" || n.Data == "article") {
			main = n
		}
	})
	target := root
	if main != nil {
		target = main
	}
	var b strings.Builder
	elements := 0
	walk(target, func(n *html.Node) bool {
		elements++
		if elements > MaxElements || b.Len() >= MaxContentRunes {
			return true
		}
		if n.Type == html.ElementNode {
			if chrome[n.Data] {
				return true
			}
			for _, a := range n.Attr {
				if a.Key == "hidden" || (a.Key == "aria-hidden" && a.Val == "true") {
					return true
				}
			}
		}
		return false
	}, func(n *html.Node) {
		if n.Type == html.TextNode && elements <= MaxElements && b.Len() < MaxContentRunes {
			text := n.Data
			if len(text) > MaxContentRunes-b.Len() {
				text = text[:MaxContentRunes-b.Len()]
			}
			b.WriteString(text)
			if b.Len() < MaxContentRunes {
				b.WriteByte(' ')
			}
		}
	})
	r.Text = strings.ToValidUTF8(collapseSpaces(b.String()), "")
	r.Tokens = len(terms.Tokenize(r.Text))
	r.NoText = r.Tokens == 0
	var total int
	walk(root, nil, func(n *html.Node) {
		if n.Type == html.TextNode {
			total += len(n.Data)
		}
	})
	r.Boilerplate = total == 0 || float64(len(r.Text))/float64(total) < BoilerplateRatio
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
