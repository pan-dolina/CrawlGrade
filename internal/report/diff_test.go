package report

import (
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

func TestCompareEqual(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	f := makeFinding(seo, "https://example.com/a")
	current := buildReport(map[string][]findings.Finding{GroupMetadata: {f}})
	baseline := buildReport(map[string][]findings.Finding{GroupMetadata: {f}})

	diff, err := Compare(current, baseline)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !diff.Equal() {
		t.Errorf("diff not equal: %+v", diff)
	}
}

func TestCompareNewAndResolved(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	old := makeFinding(seo, "https://example.com/old")
	newF := makeFinding(seo, "https://example.com/new")

	current := buildReport(map[string][]findings.Finding{GroupMetadata: {newF}})
	baseline := buildReport(map[string][]findings.Finding{GroupMetadata: {old}})

	diff, err := Compare(current, baseline)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(diff.New) != 1 || diff.New[0].URL != "https://example.com/new" {
		t.Errorf("New = %+v, want one finding at /new", diff.New)
	}
	if len(diff.Resolved) != 1 || diff.Resolved[0].URL != "https://example.com/old" {
		t.Errorf("Resolved = %+v, want one finding at /old", diff.Resolved)
	}
}

func TestCompareAddedAndRemovedGroups(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	current := buildReport(map[string][]findings.Finding{
		GroupMetadata:   {makeFinding(seo, "https://example.com/a")},
		GroupWebHygiene: {makeFinding(seo, "https://example.com/a")},
	})
	baseline := buildReport(map[string][]findings.Finding{
		GroupMetadata: {makeFinding(seo, "https://example.com/a")},
	})

	diff, err := Compare(current, baseline)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(diff.AddedGroups) != 1 || diff.AddedGroups[0] != GroupWebHygiene {
		t.Errorf("AddedGroups = %v, want [%s]", diff.AddedGroups, GroupWebHygiene)
	}
	if len(diff.RemovedGroups) != 0 {
		t.Errorf("RemovedGroups = %v, want empty", diff.RemovedGroups)
	}
}

func TestCompareSchemaMismatch(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	current := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/a")}})
	baseline := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/a")}})
	baseline.Version = "999"

	if _, err := Compare(current, baseline); err == nil {
		t.Error("Compare with schema mismatch: nil error, want error")
	}
}

func TestDiffRenderTerminal(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	current := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/new")}})
	baseline := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/old")}})

	diff, err := Compare(current, baseline)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	var b strings.Builder
	diff.RenderTerminal(&b)
	out := b.String()
	if !strings.Contains(out, "new finding(s)") {
		t.Errorf("diff output missing new findings header:\n%s", out)
	}
	if !strings.Contains(out, "resolved") {
		t.Errorf("diff output missing resolved:\n%s", out)
	}
}

func TestDiffRenderTerminalEqual(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	current := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/a")}})
	baseline := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/a")}})

	diff, err := Compare(current, baseline)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	var b strings.Builder
	diff.RenderTerminal(&b)
	if !strings.Contains(b.String(), "No changes since the baseline.") {
		t.Errorf("equal diff output:\n%s", b.String())
	}
}

func TestLoadRoundTrip(t *testing.T) {
	seo := findings.Rule{ID: "SEO-0001-001", Category: findings.CategoryMetadata, Severity: findings.SeverityLow, Title: "x"}
	rep := buildReport(map[string][]findings.Finding{GroupMetadata: {makeFinding(seo, "https://example.com/a")}})

	data, err := RenderJSON(rep)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Version != rep.Version {
		t.Errorf("loaded version = %q, want %q", loaded.Version, rep.Version)
	}
	if loaded.Summary.StartURL != rep.Summary.StartURL {
		t.Errorf("loaded start URL = %q, want %q", loaded.Summary.StartURL, rep.Summary.StartURL)
	}
}
