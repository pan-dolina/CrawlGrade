package duplicates

import (
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/content"
)

func mkPage(url, text string) *content.Result {
	return &content.Result{URL: url, Text: text}
}

func TestAnalyzeExactDuplicates(t *testing.T) {
	text := `<h1>Analiza archiwum</h1><p>Analiza archiwum to pierwszy krok do dobrze dobranych okularów.</p>`
	pages := []*Page{
		NewPage(mkPage("https://example.com/a", text)),
		NewPage(mkPage("https://example.com/b", text)),
		NewPage(mkPage("https://example.com/c", "Completely different text here.")),
	}
	r := Analyze(pages)
	if len(r.Exact) != 1 {
		t.Fatalf("expected 1 exact group, got %d", len(r.Exact))
	}
	g := r.Exact[0]
	if g.Representative != "https://example.com/a" {
		t.Fatalf("representative = %q", g.Representative)
	}
	if len(g.Duplicates) != 1 || g.Duplicates[0] != "https://example.com/b" {
		t.Fatalf("duplicates = %v", g.Duplicates)
	}
}

func TestAnalyzeNearDuplicates(t *testing.T) {
	base := `<h1>Archiwum korekcyjne</h1><p>Analiza archiwum to pierwszy krok do dobrze dobranych archiwów korekcyjnych. Analiza archiwum pomaga porządkować archiwum korekcyjne.</p>`
	// One word changed.
	alt := `<h1>Archiwum korekcyjne</h1><p>Analiza archiwum to pierwszy krok do dobrze dobranych archiwów korekcyjnych. Analiza archiwum pomaga porządkować archiwum historyczne.</p>`
	pages := []*Page{
		NewPage(mkPage("https://example.com/a", base)),
		NewPage(mkPage("https://example.com/b", alt)),
	}
	r := Analyze(pages)
	if len(r.Near) == 0 {
		t.Fatalf("expected near-duplicate group, got %+v", r.Near)
	}
}

func TestAnalyzeDistinct(t *testing.T) {
	pages := []*Page{
		NewPage(mkPage("https://example.com/a", `<h1>Archiwum korekcyjne</h1><p>Analiza archiwum.</p>`)),
		NewPage(mkPage("https://example.com/b", `<h1>Oprawki tytanowe</h1><p>Zupełnie inny temat strony.</p>`)),
	}
	r := Analyze(pages)
	if len(r.Exact) != 0 || len(r.Near) != 0 {
		t.Fatalf("expected no duplicates, got %+v", r)
	}
}

func TestFindings(t *testing.T) {
	text := `<h1>Analiza archiwum</h1><p>Analiza archiwum to pierwszy krok.</p>`
	pages := []*Page{
		NewPage(mkPage("https://example.com/a", text)),
		NewPage(mkPage("https://example.com/b", text)),
	}
	r := Analyze(pages)
	fs := r.Findings()
	if len(fs) != 1 || fs[0].ID != "SEO-DUPLICATE-001" {
		t.Fatalf("unexpected findings: %+v", fs)
	}
}
