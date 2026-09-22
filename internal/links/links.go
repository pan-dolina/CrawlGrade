// Package links builds the internal link graph of a crawl and reports the
// structural problems that only become visible across all pages: orphans,
// self links, broken internal targets and anchors that are reused for
// different targets across the site.
//
// The analysis is a pure function of the pages the crawler already visited
// and the links each page contained, so it is deterministic and testable
// without a network.
package links

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Page is the information the link graph needs from one crawled page. It is
// satisfied by *crawler.Page, whose Data carries the *htmlcheck.Page with
// the extracted links, or by the htmlcheck.Page alone when the crawl hook is
// unavailable.
type Page struct {
	URL string
	// Links are the links the page contained, already resolved.
	Links []Link
	// Indexable reports whether the page may be indexed. Orphan checks skip
	// pages that are noindexed.
	Indexable bool
	Status    int
	Failed    bool
	Start     bool
}

// Link is one link the page contained.
type Link struct {
	URL      string // resolved absolute URL; empty when unresolved
	External bool   // points outside the audited site
	Resource bool   // names a non-HTML resource
	// AnchorText is the visible text of the link, lower-cased and trimmed.
	AnchorText string
	// Empty reports whether the anchor carried no usable text.
	Empty    bool
	NoFollow bool
}

// Result holds the link-graph findings of a crawl.
type Result struct {
	findings []findings.Finding
}

// Findings returns the link-graph findings, sorted.
func (r *Result) Findings() []findings.Finding {
	return r.findings
}

// Analyze builds the link graph from pages and returns its findings.
//
// The graph is directed: an edge from A to B exists when A links to B. A
// page is an orphan when no crawled page links to it; it is a self link when
// a page links to itself.
func Analyze(pages []*Page) *Result {
	r := &Result{}

	fetched := map[string]bool{}
	failed := map[string]bool{}
	for _, p := range pages {
		if p.URL != "" {
			fetched[p.URL] = true
			failed[p.URL] = p.Failed || p.Status >= 400
		}
	}

	// inbound counts how many distinct pages link to each target.
	inbound := map[string]int{}
	// anchorTargets records, per anchor text, the distinct targets reached
	// with that text across the whole crawl.
	anchorTargets := map[string]map[string]bool{}
	// broken records internal targets that were linked to but not fetched.
	broken := map[string]bool{}

	for _, p := range pages {
		if p.URL == "" {
			continue
		}
		seen := map[string]bool{}
		for _, l := range p.Links {
			if l.URL == "" || seen[l.URL] {
				continue
			}
			seen[l.URL] = true
			if l.External {
				continue
			}
			if fetched[l.URL] {
				if l.URL != p.URL && !l.NoFollow && p.Indexable {
					inbound[l.URL]++
				}
				if failed[l.URL] {
					broken[l.URL] = true
				}
				if p.Indexable && strings.TrimSpace(l.AnchorText) != "" {
					anchorTargets[strings.ToLower(l.AnchorText)] = addTarget(anchorTargets[strings.ToLower(l.AnchorText)], l.URL)
				}
				continue
			}
			if l.Resource {
				continue
			}
			// Unvisited targets are unknown, not broken.
		}
	}

	// Orphans: indexable pages not linked to by anyone.
	for _, p := range pages {
		if p.URL != "" && p.Indexable && !p.Start && inbound[p.URL] == 0 {
			r.append(findings.LinkOrphan.New(p.URL))
		}
	}

	// Self links, per page.
	for _, p := range pages {
		if p.URL == "" {
			continue
		}
		var self []string
		for _, l := range p.Links {
			if l.URL == p.URL {
				self = append(self, p.URL)
			}
		}
		if len(self) > 0 {
			r.append(findings.LinkSelf.New(p.URL, self...))
		}
	}

	// Broken internal targets, reported once per target.
	var list []string
	for t := range broken {
		list = append(list, t)
	}
	slices.Sort(list)
	for _, t := range list {
		r.append(findings.LinkBroken.New("", t))
	}

	// Repeated anchors: the same anchor text reaching different targets.
	for text, targets := range anchorTargets {
		if len(targets) < 2 {
			continue
		}
		var tlist []string
		for t := range targets {
			tlist = append(tlist, t)
		}
		slices.Sort(tlist)
		r.append(findings.LinkDuplicated.New("",
			"anchor "+quote(text)+" points to "+strconv.Itoa(len(tlist))+" pages"))
	}

	r.sort()
	return r
}

// addTarget records that text reaches target.
func addTarget(m map[string]bool, target string) map[string]bool {
	if m == nil {
		m = map[string]bool{}
	}
	m[target] = true
	return m
}

func (r *Result) append(f findings.Finding) {
	r.findings = append(r.findings, f)
}

func (r *Result) sort() {
	findings.Sort(r.findings)
}

// quote wraps s in double quotes for display in evidence.
func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
