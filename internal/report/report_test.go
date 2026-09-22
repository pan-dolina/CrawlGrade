package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/terms"
)

// makeFinding builds a finding from a rule and URL.
func makeFinding(rule findings.Rule, u string, evidence ...string) findings.Finding {
	return rule.New(u, evidence...)
}

// sampleCrawl builds a minimal crawler.Result with one page.
func sampleCrawl() *crawler.Result {
	return &crawler.Result{
		Pages:        []*crawler.Page{},
		SkippedCount: map[string]int{},
		StopReason:   crawler.StopCompleted,
	}
}

// buildReport constructs a report from grouped findings.
func buildReport(groups map[string][]findings.Finding) *Report {
	return New("https://example.com/", sampleCrawl(), groups)
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"":         FormatTerminal,
		"terminal": FormatTerminal,
		"json":     FormatJSON,
		"html":     FormatHTML,
		"JSON":     FormatJSON,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil {
			t.Fatalf("ParseFormat(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("ParseFormat(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml) = nil error, want error")
	}
}

func TestRenderJSON(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "Duplicate title"}
	rep := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(seo, "https://example.com/a")},
	})

	var buf bytes.Buffer
	if err := rep.Render(&buf, FormatJSON); err != nil {
		t.Fatalf("Render(json): %v", err)
	}
	out := buf.Bytes()

	// The output must be valid JSON with the schema version.
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["schema_version"] != "1" {
		t.Errorf("schema_version = %v, want 1", parsed["schema_version"])
	}

	// The output must be indented deterministically.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, out, "", "  "); err != nil {
		t.Fatalf("Indent: %v", err)
	}
	if pretty.String() != string(out) {
		t.Error("JSON output is not indented")
	}
}

func TestRenderTerminal(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "Duplicate title"}
	rep := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(seo, "https://example.com/a", "title: Home")},
	})

	var buf bytes.Buffer
	if err := RenderTerminal(&buf, rep); err != nil {
		t.Fatalf("RenderTerminal: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "https://example.com/") {
		t.Error("terminal output missing start URL")
	}
	if !strings.Contains(out, "Duplicate title") {
		t.Error("terminal output missing finding title")
	}
	if !strings.Contains(out, "Metadata") {
		t.Error("terminal output missing category name")
	}
}

func TestRenderTerminalEmpty(t *testing.T) {
	rep := buildReport(nil)
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, rep); err != nil {
		t.Fatalf("RenderTerminal: %v", err)
	}
	if !strings.Contains(buf.String(), "No findings.") {
		t.Error("empty report should report No findings.")
	}
}

func TestRenderTerminalCompactAndDetailed(t *testing.T) {
	f := findings.TitleMissing.New("https://example.com/", "evidence line")
	rep := buildReport(map[string][]findings.Finding{GroupMetadata: {f}})
	var compact, detailed bytes.Buffer
	if err := RenderTerminal(&compact, rep); err != nil {
		t.Fatal(err)
	}
	if err := RenderTerminalDetailed(&detailed, rep); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compact.String(), "evidence line") || strings.Contains(compact.String(), f.Recommendation) {
		t.Fatalf("compact output contains details:\n%s", compact.String())
	}
	if !strings.Contains(detailed.String(), "evidence line") {
		t.Fatalf("detailed output omitted evidence:\n%s", detailed.String())
	}
}

func TestRenderHTMLQuickWins(t *testing.T) {
	f := findings.Rule{ID: "SEO-TEST-001", Category: findings.CategoryMetadata, Severity: findings.SeverityHigh, Title: "Fix this", Recommendation: "Do this."}.New("https://example.com/", "evidence")
	rep := buildReport(map[string][]findings.Finding{GroupMetadata: {f}})
	var b bytes.Buffer
	if err := RenderHTMLQuickWins(&b, rep); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "Quick wins") || !strings.Contains(b.String(), "Fix this") || strings.Contains(b.String(), "No findings.") {
		t.Fatal(b.String())
	}
	if !strings.Contains(b.String(), "Instrukcje dla LLM") || !strings.Contains(b.String(), "Instrukcja dla LLM") {
		t.Errorf("quick-wins HTML missing LLM instructions:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "sev-border-high") {
		t.Errorf("quick-wins HTML missing high contrast severity border class:\n%s", b.String())
	}
	if strings.Contains(b.String(), "href=\"#findings-") {
		t.Errorf("quick-wins mode should not link to absent findings sections:\n%s", b.String())
	}
}

func TestRenderHTML(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "Duplicate title"}
	rep := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(seo, "https://example.com/a")},
	})

	var buf bytes.Buffer
	if err := rep.Render(&buf, FormatHTML); err != nil {
		t.Fatalf("Render(html): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Duplicate title") {
		t.Error("HTML output missing finding title")
	}
	// The HTML must be escaped: a title with markup must not appear raw.
	esc := findings.Rule{ID: "SEO-0001-002", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "<script>alert(1)</script>"}
	rep2 := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(esc, "https://example.com/a")},
	})
	var buf2 bytes.Buffer
	if err := rep2.Render(&buf2, FormatHTML); err != nil {
		t.Fatalf("Render(html): %v", err)
	}
	if strings.Contains(buf2.String(), "<script>alert(1)</script>") {
		t.Error("HTML output is not escaped")
	}
	if !strings.Contains(buf2.String(), "&lt;script&gt;") {
		t.Error("HTML output is not escaped")
	}
}

func TestRenderHTMLPrioritizesTermsAndCollapsesDetails(t *testing.T) {
	critical := findings.Rule{ID: "SEO-TEST-CRITICAL", Category: findings.CategoryMetadata, Severity: findings.SeverityCritical, Title: "Critical issue", Recommendation: "Fix it."}.New("https://example.com/")
	low := findings.Rule{ID: "SEO-TEST-LOW", Category: findings.CategoryContent, Severity: findings.SeverityLow, Title: "Low issue", Recommendation: "Review it."}.New("https://example.com/about")
	rep := buildReport(map[string][]findings.Finding{
		GroupMetadata: {critical},
		GroupContent:  {low},
	})
	rep.Terms = []terms.Score{{Term: "strong-term", Strength: 1}}

	var buf bytes.Buffer
	if err := rep.Render(&buf, FormatHTML); err != nil {
		t.Fatalf("Render(html): %v", err)
	}
	out := buf.String()
	termsAt := strings.Index(out, "Najsilniejsze hasła")
	quickWinsAt := strings.Index(out, "Quick wins")
	detailsAt := strings.Index(out, "Findingi według priorytetu")
	if termsAt < 0 || quickWinsAt < 0 || detailsAt < 0 {
		t.Fatalf("HTML missing prioritized sections:\n%s", out)
	}
	if !(termsAt < quickWinsAt && quickWinsAt < detailsAt) {
		t.Fatalf("HTML sections are out of order: terms=%d quick-wins=%d details=%d", termsAt, quickWinsAt, detailsAt)
	}
	if !strings.Contains(out, "<details") || !strings.Contains(out, "Critical issue") || !strings.Contains(out, "Low issue") {
		t.Fatalf("HTML missing collapsible finding details:\n%s", out)
	}
	for _, want := range []string{
		"href=\"#findings-critical\"",
		"id=\"findings-critical\"",
		"href=\"#findings-low\"",
		"id=\"findings-low\"",
		"id=\"terms\"",
		"id=\"quick-wins\"",
		"id=\"llm-instructions\"",
		"Instrukcja dla LLM",
		"Działaj jako senior web developer",
		"href=\"#top\"",
		"class=\"to-top\"",
		"Do góry",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML missing navigation element %q", want)
		}
	}
	if strings.Contains(out, "href=\"#findings-high\"") {
		t.Errorf("HTML contains link to zero-count severity: findings-high")
	}
}

func TestCategoryGroups(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	rep := buildReport(map[string][]findings.Finding{
		GroupWebHygiene: {makeFinding(seo, "https://example.com/a")},
	})
	got := rep.CategoryGroups()
	if len(got) != 1 || got[0] != GroupWebHygiene {
		t.Errorf("CategoryGroups = %v, want [%s]", got, GroupWebHygiene)
	}
}

func TestFindingCountAndFindings(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	rep := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(seo, "https://example.com/a")},
	})
	if n := rep.FindingCount(GroupMetadata); n != 1 {
		t.Errorf("FindingCount(metadata) = %d, want 1", n)
	}
	if n := rep.FindingCount(GroupCrawl); n != 0 {
		t.Errorf("FindingCount(crawlability) = %d, want 0", n)
	}
	if got := rep.Findings(GroupMetadata); len(got) != 1 {
		t.Errorf("Findings(metadata) len = %d, want 1", len(got))
	}
}
