package links

import (
	"slices"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

func ids(fs []findings.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.ID)
	}
	return strings.Join(s, ",")
}

func page(url string, indexable bool, links ...Link) *Page {
	return &Page{URL: url, Indexable: indexable, Links: links}
}

func TestOrphan(t *testing.T) {
	// a links to b but nothing links to a; c is never linked. Both a and c
	// are orphans.
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://example.com/b"}),
		page("https://example.com/b", true),
		page("https://example.com/c", true),
	})
	got := ids(r.Findings())
	if got != "SEO-LINK-001,SEO-LINK-001" {
		t.Fatalf("orphans = %s", got)
	}
	urls := map[string]bool{}
	for _, f := range r.Findings() {
		urls[f.URL] = true
	}
	if !urls["https://example.com/a"] || !urls["https://example.com/c"] {
		t.Errorf("orphans = %v", urls)
	}
}

func TestLinkedPageIsNotOrphan(t *testing.T) {
	// b is linked from a, so only a is an orphan.
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://example.com/b"}),
		page("https://example.com/b", true),
	})
	if ids(r.Findings()) != "SEO-LINK-001" {
		t.Fatalf("orphans = %s", ids(r.Findings()))
	}
}

func TestSelfLink(t *testing.T) {
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://example.com/a"}),
	})
	if ids(r.Findings()) != "SEO-LINK-002" {
		t.Fatalf("self = %s", ids(r.Findings()))
	}
}

func TestBrokenInternalTarget(t *testing.T) {
	// a links to /missing, which was not fetched, and to /img.png, a
	// resource the crawler checked with HEAD (not broken). a is an orphan
	// because nothing links to it.
	r := Analyze([]*Page{
		page("https://example.com/a", true,
			Link{URL: "https://example.com/missing"},
			Link{URL: "https://example.com/img.png", Resource: true},
		),
	})
	got := ids(r.Findings())
	if got != "SEO-LINK-004,SEO-LINK-001" {
		t.Fatalf("broken = %s", got)
	}
	var found bool
	for _, f := range r.Findings() {
		if f.ID == "SEO-LINK-004" && slices.Contains(f.Evidence, "https://example.com/missing") {
			found = true
		}
	}
	if !found {
		t.Errorf("broken evidence missing: %v", r.Findings())
	}
}

func TestRepeatedAnchorAcrossSite(t *testing.T) {
	// "frames" points at /frames-a on one page and /frames-b on another.
	// a and b are orphans (nothing links to them).
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://example.com/frames-a", AnchorText: "frames"}),
		page("https://example.com/b", true, Link{URL: "https://example.com/frames-b", AnchorText: "Frames"}),
		page("https://example.com/frames-a", true),
		page("https://example.com/frames-b", true),
	})
	got := ids(r.Findings())
	// The two orphans (medium) sort before the repeated-anchor finding
	// (info).
	if got != "SEO-LINK-001,SEO-LINK-001,SEO-LINK-003" {
		t.Fatalf("repeated anchor = %s", got)
	}
}

func TestSameAnchorSameTargetIsNotReported(t *testing.T) {
	// Both pages link to /frames with the same anchor; the target is the
	// same, so there is no ambiguity. a and b are orphans.
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://example.com/frames", AnchorText: "frames"}),
		page("https://example.com/b", true, Link{URL: "https://example.com/frames", AnchorText: "frames"}),
		page("https://example.com/frames", true),
	})
	got := ids(r.Findings())
	if got != "SEO-LINK-001,SEO-LINK-001" {
		t.Fatalf("findings = %s", got)
	}
}

func TestExternalLinksIgnored(t *testing.T) {
	// a links externally to other.example; a is an orphan (nothing links to
	// it). The external link is not a broken internal target.
	r := Analyze([]*Page{
		page("https://example.com/a", true, Link{URL: "https://other.example/x", External: true}),
	})
	if ids(r.Findings()) != "SEO-LINK-001" {
		t.Fatalf("findings = %s", ids(r.Findings()))
	}
}
