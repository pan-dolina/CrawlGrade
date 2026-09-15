// Package structureddata extracts JSON-LD and microdata from HTML and runs
// basic, documented checks for common schema.org types.
//
// The checks cover syntax and a small set of properties per type. They are
// not a reimplementation of any search engine's rich result validation, and
// a page without findings is not guaranteed to be eligible for rich results.
package structureddata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Limits on structured data processed per page.
const (
	MaxBlockBytes = 1 << 20
	MaxBlocks     = 50
	MaxNodes      = 10000
	MaxDepth      = 32
)

// Item is a typed JSON-LD node.
type Item struct {
	Block int            `json:"block"`
	Path  string         `json:"path"`
	Types []string       `json:"types"`
	Props map[string]any `json:"-"`
	// TopLevel is true for nodes at the root of a block or of @graph.
	TopLevel bool `json:"-"`
}

// Is reports whether the item has type t.
func (it Item) Is(t string) bool { return slices.Contains(it.Types, t) }

// BlockError describes a JSON-LD block that could not be used.
type BlockError struct {
	Block   int    `json:"block"`
	Message string `json:"message"`
}

// Result holds the structured data of one page.
type Result struct {
	Blocks        int          `json:"jsonld_blocks"`
	Items         []Item       `json:"items,omitempty"`
	Errors        []BlockError `json:"errors,omitempty"`
	Untyped       []string     `json:"-"`
	ContextIssues []string     `json:"-"`
	Truncated     bool         `json:"truncated,omitempty"`
}

// Types returns the distinct schema.org types found, sorted.
func (r *Result) Types() []string {
	var out []string
	for _, it := range r.Items {
		out = append(out, it.Types...)
	}
	sort.Strings(out)
	return slices.Compact(out)
}

// Extract collects JSON-LD blocks and microdata from a parsed document.
func Extract(root *html.Node) *Result {
	r := &Result{}
	var scripts []string
	walk(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == atom.Template {
			return false
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Script && n.Namespace == "" && isJSONLD(attr(n, "type")) {
			var b strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					b.WriteString(c.Data)
				}
			}
			scripts = append(scripts, b.String())
			return false
		}
		return true
	})
	for i, s := range scripts {
		if i >= MaxBlocks {
			r.Truncated = true
			break
		}
		r.Blocks++
		r.parseBlock(i+1, s)
	}
	return r
}

func isJSONLD(typ string) bool {
	mt, _, _ := strings.Cut(typ, ";")
	return strings.EqualFold(strings.TrimSpace(mt), "application/ld+json")
}

func (r *Result) parseBlock(block int, src string) {
	if len(src) > MaxBlockBytes {
		r.Errors = append(r.Errors, BlockError{block, fmt.Sprintf("block larger than %d bytes", MaxBlockBytes)})
		r.Truncated = true
		return
	}
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		r.Errors = append(r.Errors, BlockError{block, "empty block"})
		return
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		r.Errors = append(r.Errors, BlockError{block, "invalid JSON: " + describeJSONError(err, trimmed)})
		return
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		r.Errors = append(r.Errors, BlockError{block, "invalid JSON: unexpected data after the top-level value"})
		return
	}
	w := walker{r: r, block: block}
	switch t := v.(type) {
	case map[string]any:
		w.top(t, "")
	case []any:
		for i, e := range t {
			if m, ok := e.(map[string]any); ok {
				w.top(m, fmt.Sprintf("[%d]", i))
			}
		}
	default:
		r.Errors = append(r.Errors, BlockError{block, "top-level value is not an object or array"})
	}
}

func describeJSONError(err error, src string) string {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		line := 1 + bytes.Count([]byte(src[:min(int(se.Offset), len(src))]), []byte("\n"))
		return fmt.Sprintf("%s (line %d)", se.Error(), line)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "unexpected end of input"
	}
	return err.Error()
}

type walker struct {
	r     *Result
	block int
	nodes int
}

func (w *walker) top(m map[string]any, path string) {
	ctx, hasCtx := m["@context"]
	if graph, ok := m["@graph"].([]any); ok {
		if !hasCtx {
			w.r.ContextIssues = append(w.r.ContextIssues, fmt.Sprintf("block %d%s: @graph without @context", w.block, path))
		} else if !schemaContext(ctx) {
			w.r.ContextIssues = append(w.r.ContextIssues, fmt.Sprintf("block %d%s: @context is not schema.org", w.block, path))
		}
		for i, e := range graph {
			if em, ok := e.(map[string]any); ok {
				w.node(em, fmt.Sprintf("%s@graph[%d]", prefix(path), i), 0, true)
			}
		}
		return
	}
	switch {
	case !hasCtx:
		w.r.ContextIssues = append(w.r.ContextIssues, fmt.Sprintf("block %d%s: missing @context", w.block, path))
	case !schemaContext(ctx):
		w.r.ContextIssues = append(w.r.ContextIssues, fmt.Sprintf("block %d%s: @context is not schema.org", w.block, path))
	}
	w.node(m, strings.TrimPrefix(path, " > "), 0, true)
}

func prefix(path string) string {
	if path == "" {
		return ""
	}
	return path + " > "
}

func (w *walker) node(m map[string]any, path string, depth int, top bool) {
	w.nodes++
	if w.nodes > MaxNodes || depth > MaxDepth {
		w.r.Truncated = true
		return
	}
	types := typesOf(m["@type"])
	if len(types) > 0 {
		w.r.Items = append(w.r.Items, Item{Block: w.block, Path: displayPath(path), Types: types, Props: m, TopLevel: top})
	} else if top {
		if _, isRef := m["@id"]; !isRef || len(m) > 2 {
			w.r.Untyped = append(w.r.Untyped, fmt.Sprintf("block %d %s", w.block, displayPath(path)))
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "@context" || k == "@graph" {
			continue
		}
		w.value(m[k], prefix(path)+k, depth+1)
	}
}

func (w *walker) value(v any, path string, depth int) {
	switch t := v.(type) {
	case map[string]any:
		w.node(t, path, depth, false)
	case []any:
		for i, e := range t {
			w.value(e, fmt.Sprintf("%s[%d]", path, i), depth+1)
			if w.nodes > MaxNodes {
				return
			}
		}
	}
}

func displayPath(p string) string {
	if p == "" {
		return "(root)"
	}
	return p
}

func schemaContext(ctx any) bool {
	switch t := ctx.(type) {
	case string:
		s := strings.TrimRight(strings.ToLower(strings.TrimSpace(t)), "/")
		return s == "https://schema.org" || s == "http://schema.org"
	case []any:
		for _, e := range t {
			if schemaContext(e) {
				return true
			}
		}
	case map[string]any:
		if v, ok := t["@vocab"]; ok {
			return schemaContext(v)
		}
	}
	return false
}

// typesOf normalizes @type values ("Product", "schema:Product",
// "https://schema.org/Product") to local names.
func typesOf(v any) []string {
	var raw []string
	switch t := v.(type) {
	case string:
		raw = []string{t}
	case []any:
		for _, e := range t {
			if s, ok := e.(string); ok {
				raw = append(raw, s)
			}
		}
	}
	var out []string
	for _, s := range raw {
		s = strings.TrimSpace(s)
		for _, p := range []string{"https://schema.org/", "http://schema.org/", "schema:"} {
			if len(s) > len(p) && strings.EqualFold(s[:len(p)], p) {
				s = s[len(p):]
				break
			}
		}
		if s != "" && len(s) <= 100 && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

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

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val
		}
	}
	return ""
}
