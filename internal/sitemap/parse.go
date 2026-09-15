// Package sitemap fetches and parses XML sitemaps and sitemap indexes
// (sitemaps.org protocol 0.9).
//
// Parsing is streaming and bounded: documents are read through a size
// limit, entries beyond the protocol's 50 000 per file are ignored, and
// encoding/xml does not expand external or custom entities, so XML entity
// expansion attacks do not apply.
package sitemap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Protocol limits.
const (
	MaxEntriesPerFile = 50000
	MaxLocLength      = 2048
	maxWarnings       = 20
	Namespace         = "http://www.sitemaps.org/schemas/sitemap/0.9"
)

// Kind is the root element type.
type Kind string

// Document kinds.
const (
	KindURLSet       Kind = "urlset"
	KindSitemapIndex Kind = "sitemapindex"
)

// ErrNotSitemap is returned for XML documents with another root element.
var ErrNotSitemap = errors.New("not a sitemap: root element must be urlset or sitemapindex")

// Entry is a <url> or <sitemap> element.
type Entry struct {
	Loc     string `json:"loc"`
	LastMod string `json:"lastmod,omitempty"`
}

// Document is a parsed sitemap file.
type Document struct {
	Kind      Kind
	Entries   []Entry
	Warnings  []string
	Truncated bool // more than MaxEntriesPerFile entries
}

func (d *Document) warn(format string, args ...any) {
	if len(d.Warnings) < maxWarnings {
		d.Warnings = append(d.Warnings, fmt.Sprintf(format, args...))
	}
}

type rawEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// Parse reads a sitemap or sitemap index from r. The caller bounds the size
// of r.
func Parse(r io.Reader) (*Document, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	// Only UTF-8 (and its ASCII subset) is supported; other declared
	// encodings are rejected by CharsetReader being nil.
	doc := &Document{}
	var entryName string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			if doc.Kind == "" {
				return nil, fmt.Errorf("%w: empty document", ErrNotSitemap)
			}
			return doc, nil
		}
		if err != nil {
			return doc, fmt.Errorf("invalid XML: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if doc.Kind == "" {
			switch se.Name.Local {
			case string(KindURLSet):
				doc.Kind, entryName = KindURLSet, "url"
			case string(KindSitemapIndex):
				doc.Kind, entryName = KindSitemapIndex, "sitemap"
			default:
				return nil, fmt.Errorf("%w (found %q)", ErrNotSitemap, truncate(se.Name.Local, 40))
			}
			if se.Name.Space != Namespace {
				doc.warn("root element namespace is %q, expected %q", truncate(se.Name.Space, 80), Namespace)
			}
			continue
		}
		if se.Name.Local != entryName {
			// Extensions (image:image, xhtml:link, ...) and unknown elements.
			if err := dec.Skip(); err != nil {
				return doc, fmt.Errorf("invalid XML: %w", err)
			}
			continue
		}
		if len(doc.Entries) >= MaxEntriesPerFile {
			doc.Truncated = true
			if err := dec.Skip(); err != nil {
				return doc, fmt.Errorf("invalid XML: %w", err)
			}
			continue
		}
		var e rawEntry
		if err := dec.DecodeElement(&e, &se); err != nil {
			return doc, fmt.Errorf("invalid XML: %w", err)
		}
		loc := strings.TrimSpace(e.Loc)
		switch {
		case loc == "":
			doc.warn("<%s> without <loc>", entryName)
			continue
		case len(loc) > MaxLocLength:
			doc.warn("<loc> longer than %d bytes", MaxLocLength)
			continue
		}
		lastmod := strings.TrimSpace(e.LastMod)
		if lastmod != "" && !ValidLastMod(lastmod) {
			doc.warn("invalid <lastmod> %q for %s", truncate(lastmod, 40), truncate(loc, 200))
		}
		doc.Entries = append(doc.Entries, Entry{Loc: loc, LastMod: lastmod})
	}
}

var lastModLayouts = []string{
	"2006-01-02",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15:04:05Z07:00",
	time.RFC3339Nano,
	"2006-01",
	"2006",
}

// ValidLastMod reports whether s is a W3C Datetime value.
func ValidLastMod(s string) bool {
	for _, l := range lastModLayouts {
		if _, err := time.Parse(l, s); err == nil {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "") + "..."
}
