package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/findings"
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
