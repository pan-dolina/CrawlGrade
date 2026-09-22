package terms_strength

import (
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/content"
	"github.com/pan-dolina/crawlgrade/internal/findings"
)

func mkResult(url, text string) *content.Result {
	return &content.Result{URL: url, Text: text}
}

func TestPageSpecificTerm(t *testing.T) {
	pages := []*content.Result{
		mkResult("https://example.com/a", `<h1>Analiza archiwum</h1><p>Analiza archiwum to pierwszy krok do dobrze dobranych okularów.</p>`),
		mkResult("https://example.com/b", `<h1>Oprawki tytanowe</h1><p>Oprawki tytanowe są lekkie i trwałe.</p>`),
	}
	r := Analyze(pages)
	fs := r.Findings()
	// "analiza archiwum" is page-specific to /a.
	if !hasFindingID(fs, "SEO-TERMS-002") {
		t.Fatalf("expected page-specific term finding, got %+v", fs)
	}
}

func TestSiteWideTerm(t *testing.T) {
	// A term repeated across many pages.
	text := `<h1>Archiwum korekcyjne</h1><p>Archiwum korekcyjne są wygodne.</p>`
	var pages []*content.Result
	for _, u := range []string{"/a", "/b", "/c"} {
		pages = append(pages, mkResult("https://example.com"+u, text))
	}
	r := Analyze(pages)
	if len(r.SiteWide) == 0 {
		t.Fatalf("expected site-wide term, got %+v", r.SiteWide)
	}
}

func TestDuplicateProfile(t *testing.T) {
	text := `<h1>Analiza archiwum</h1><p>Analiza archiwum to pierwszy krok do dobrze dobranych okularów korekacyjnych.</p>`
	pages := []*content.Result{
		mkResult("https://example.com/a", text),
		mkResult("https://example.com/b", text),
	}
	r := Analyze(pages)
	if len(r.DuplicateProfile) == 0 {
		t.Fatalf("expected duplicate profile, got %+v", r.DuplicateProfile)
	}
}

func hasFindingID(fs []findings.Finding, id string) bool {
	for _, f := range fs {
		if f.ID == id {
			return true
		}
	}
	return false
}
