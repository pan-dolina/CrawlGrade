package sitemap

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

// Limits bound sitemap discovery.
type Limits struct {
	MaxFiles int   // sitemap files fetched, including indexes
	MaxURLs  int   // page URLs kept across all files
	MaxBytes int64 // per file, after decompression
	MaxDepth int   // index nesting; 1 allows an index of sitemaps
}

// DefaultLimits returns the documented defaults.
func DefaultLimits() Limits {
	return Limits{MaxFiles: 50, MaxURLs: 50000, MaxBytes: 50 << 20, MaxDepth: 2}
}

// Fetcher performs requests; *fetcher.Client implements it.
type Fetcher interface {
	Fetch(ctx context.Context, req fetcher.Request) (*fetcher.Response, error)
}

// File describes one fetched sitemap file.
type File struct {
	URL      string   `json:"url"`
	FinalURL string   `json:"final_url,omitempty"`
	Source   string   `json:"source"` // robots.txt, default, or the parent index URL
	Status   int      `json:"status,omitempty"`
	Kind     Kind     `json:"kind,omitempty"`
	Entries  int      `json:"entries"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	// Truncated is set when the file had more entries than the protocol
	// allows or than MaxURLs left room for.
	Truncated bool `json:"truncated,omitempty"`
	Depth     int  `json:"-"`
	// Declared is true for files listed in robots.txt or given explicitly;
	// the default /sitemap.xml probe is not declared.
	Declared bool `json:"-"`
}

// URL is a page URL listed in a sitemap.
type URL struct {
	Loc     string `json:"loc"` // normalized
	LastMod string `json:"lastmod,omitempty"`
	Sitemap string `json:"sitemap"`
}

// Result is the outcome of sitemap discovery.
type Result struct {
	Files        []*File
	URLs         []URL // sorted by Loc, unique
	OutOfScope   []string
	InvalidLocs  []string
	FilesSkipped int
	URLsSkipped  int
}

// Seed is a sitemap URL to start from.
type Seed struct {
	URL      string
	Source   string
	Declared bool
}

// Collect fetches the seeds and the sitemaps they reference, breadth-first.
func Collect(ctx context.Context, f Fetcher, seeds []Seed, scope urlnorm.Scope, limits Limits) *Result {
	d := DefaultLimits()
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = d.MaxFiles
	}
	if limits.MaxURLs <= 0 {
		limits.MaxURLs = d.MaxURLs
	}
	if limits.MaxBytes <= 0 {
		limits.MaxBytes = d.MaxBytes
	}
	if limits.MaxDepth <= 0 {
		limits.MaxDepth = d.MaxDepth
	}

	res := &Result{}
	seenFiles := map[string]bool{}
	seenURLs := map[string]bool{}
	queue := []*File{}
	enqueue := func(raw, source string, depth int, declared bool) {
		key := raw
		u, err := urlnorm.Parse(raw)
		if err == nil {
			key = u.String()
		}
		if seenFiles[key] {
			return
		}
		seenFiles[key] = true
		if len(queue)+len(res.Files) >= limits.MaxFiles {
			res.FilesSkipped++
			return
		}
		file := &File{URL: key, Source: source, Depth: depth, Declared: declared}
		if err != nil {
			file.URL = truncate(raw, 200)
			file.Error = "invalid sitemap URL: " + err.Error()
			res.Files = append(res.Files, file)
			return
		}
		queue = append(queue, file)
	}
	for _, s := range seeds {
		enqueue(s.URL, s.Source, 0, s.Declared)
	}

	for len(queue) > 0 && ctx.Err() == nil {
		file := queue[0]
		queue = queue[1:]
		res.Files = append(res.Files, file)
		doc := fetchFile(ctx, f, file, limits.MaxBytes)
		if doc == nil {
			continue
		}
		file.Kind = doc.Kind
		file.Entries = len(doc.Entries)
		file.Warnings = doc.Warnings
		file.Truncated = doc.Truncated
		switch doc.Kind {
		case KindSitemapIndex:
			if file.Depth+1 > limits.MaxDepth {
				file.Warnings = append(file.Warnings, fmt.Sprintf("sitemap index nesting deeper than %d levels is not followed", limits.MaxDepth))
				continue
			}
			if file.Depth > 0 {
				file.Warnings = append(file.Warnings, "sitemap index references another sitemap index")
			}
			for _, e := range doc.Entries {
				enqueue(e.Loc, file.URL, file.Depth+1, true)
			}
		case KindURLSet:
			for _, e := range doc.Entries {
				u, err := urlnorm.Parse(e.Loc)
				if err != nil {
					if len(res.InvalidLocs) < 100 {
						res.InvalidLocs = append(res.InvalidLocs, fmt.Sprintf("%s: %s", file.URL, truncate(e.Loc, 200)))
					}
					continue
				}
				if !scope.Contains(u) {
					if len(res.OutOfScope) < 100 {
						res.OutOfScope = append(res.OutOfScope, u.String())
					}
					continue
				}
				key := u.String()
				if seenURLs[key] {
					continue
				}
				if len(res.URLs) >= limits.MaxURLs {
					res.URLsSkipped++
					continue
				}
				seenURLs[key] = true
				res.URLs = append(res.URLs, URL{Loc: key, LastMod: e.LastMod, Sitemap: file.URL})
			}
		}
	}
	slices.SortFunc(res.URLs, func(a, b URL) int {
		if a.Loc < b.Loc {
			return -1
		}
		if a.Loc > b.Loc {
			return 1
		}
		return 0
	})
	return res
}

func fetchFile(ctx context.Context, f Fetcher, file *File, maxBytes int64) *Document {
	resp, err := f.Fetch(ctx, fetcher.Request{
		URL:                  file.URL,
		Accept:               "application/xml,text/xml;q=0.9,*/*;q=0.5",
		MaxBodyBytes:         maxBytes,
		MaxDecompressedBytes: maxBytes,
	})
	if resp != nil {
		file.Status = resp.Status
		if resp.FinalURL != file.URL {
			file.FinalURL = resp.FinalURL
		}
	}
	if err != nil {
		file.Error = err.Error()
		return nil
	}
	if resp.Status < 200 || resp.Status > 299 {
		file.Error = fmt.Sprintf("HTTP status %d", resp.Status)
		return nil
	}
	doc, err := Parse(bytes.NewReader(resp.Body))
	if err != nil {
		file.Error = err.Error()
		if doc != nil {
			file.Kind = doc.Kind
		}
		return nil
	}
	return doc
}

// Findings reports problems with the sitemap files themselves. Comparisons
// between sitemap URLs and the crawl are made by the audit.
func (r *Result) Findings() []findings.Finding {
	var out []findings.Finding
	if len(r.Files) == 0 || !r.anyUsable() {
		declared := false
		for _, f := range r.Files {
			declared = declared || f.Declared
		}
		if !declared {
			var ev []string
			for _, f := range r.Files {
				ev = append(ev, fmt.Sprintf("%s: %s", f.URL, f.Error))
			}
			out = append(out, findings.SitemapMissing.New("", ev...))
		}
	}
	for _, f := range r.Files {
		if f.Error != "" {
			switch {
			case f.Status >= 200 && f.Status < 300:
				out = append(out, findings.SitemapInvalid.New(f.URL, f.Error))
			case f.Declared:
				out = append(out, findings.SitemapUnavailable.New(f.URL, "source: "+f.Source, f.Error))
			}
			// A failing probe of the undeclared /sitemap.xml is covered by
			// SitemapMissing.
		}
		if f.Truncated {
			out = append(out, findings.SitemapTooLarge.New(f.URL, fmt.Sprintf("%d entries", f.Entries)))
		}
		if len(f.Warnings) > 0 {
			out = append(out, findings.SitemapWarnings.New(f.URL, f.Warnings...))
		}
	}
	if len(r.OutOfScope) > 0 {
		out = append(out, findings.SitemapOutOfScope.New("", r.OutOfScope...))
	}
	if len(r.InvalidLocs) > 0 {
		out = append(out, findings.SitemapInvalidLoc.New("", r.InvalidLocs...))
	}
	if r.FilesSkipped > 0 || r.URLsSkipped > 0 {
		out = append(out, findings.SitemapLimitReached.New("",
			fmt.Sprintf("sitemap files not fetched: %d", r.FilesSkipped),
			fmt.Sprintf("sitemap URLs not recorded: %d", r.URLsSkipped)))
	}
	return out
}

func (r *Result) anyUsable() bool {
	for _, f := range r.Files {
		if f.Error == "" && f.Kind != "" {
			return true
		}
	}
	return false
}

// Contains reports whether the normalized URL is listed.
func (r *Result) Contains(loc string) bool {
	_, ok := slices.BinarySearchFunc(r.URLs, loc, func(u URL, s string) int {
		switch {
		case u.Loc < s:
			return -1
		case u.Loc > s:
			return 1
		}
		return 0
	})
	return ok
}

// DefaultSeed returns the conventional /sitemap.xml location for site.
func DefaultSeed(site *url.URL) Seed {
	u := &url.URL{Scheme: site.Scheme, Host: site.Host, Path: "/sitemap.xml"}
	return Seed{URL: u.String(), Source: "default location"}
}
