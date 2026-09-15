package findings

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

var testRule = Rule{
	ID:             "SEO-TEST-001",
	Category:       CategoryMetadata,
	Severity:       SeverityMedium,
	Title:          "Test rule",
	Recommendation: "Fix it.",
}

func TestSeverityRoundTrip(t *testing.T) {
	for _, s := range Severities() {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		var got Severity
		if err := json.Unmarshal(b, &got); err != nil || got != s {
			t.Errorf("%v: got %v, %v", s, got, err)
		}
	}
	if _, err := ParseSeverity("HIGH"); err != nil {
		t.Error("severity names are case-insensitive")
	}
	if _, err := ParseSeverity("pass"); err == nil {
		t.Error("unknown severity accepted")
	}
	if _, err := json.Marshal(Severity(0)); err == nil {
		t.Error("zero severity marshalled")
	}
	if Severity(42).String() != "Severity(42)" {
		t.Error("invalid severity string")
	}
	var s Severity
	if err := s.UnmarshalText([]byte("nope")); err == nil {
		t.Error("invalid text accepted")
	}
}

func TestCategories(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Categories() {
		if !c.Valid() || c.Name() == string(c) {
			t.Errorf("category %q lacks a display name", c)
		}
		names[c.Name()] = true
	}
	for _, want := range []string{"Crawlability", "Metadata", "Content", "Structured Data", "Internal Linking", "Web Hygiene"} {
		if !names[want] {
			t.Errorf("missing category %q", want)
		}
	}
	if Category("bogus").Valid() || Category("bogus").Name() != "bogus" {
		t.Error("bogus category")
	}
}

func TestNewLimitsEvidence(t *testing.T) {
	lines := make([]string, MaxEvidenceLines+5)
	for i := range lines {
		lines[i] = strings.Repeat("ż", MaxEvidenceRunes+10)
	}
	f := testRule.New("https://example.com/", lines...)
	if len(f.Evidence) != MaxEvidenceLines+1 {
		t.Fatalf("evidence lines = %d", len(f.Evidence))
	}
	if f.Evidence[MaxEvidenceLines] != "... and 5 more" {
		t.Errorf("overflow line = %q", f.Evidence[MaxEvidenceLines])
	}
	if n := utf8.RuneCountInString(f.Evidence[0]); n != MaxEvidenceRunes {
		t.Errorf("truncated line has %d runes", n)
	}
	lines[0] = "changed"
	if f.Evidence[0] == "changed" {
		t.Error("evidence aliases caller slice")
	}
	if testRule.New("").Evidence != nil {
		t.Error("empty evidence should be nil")
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"abc", 3, "abc"},
		{"abcd", 3, "ab…"},
		{"abcd", 1, "…"},
		{"\xff\xfe", 5, "\uFFFD"},
	}
	for _, c := range cases {
		if got := Truncate(c.in, c.max); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestSortCountsAndThresholds(t *testing.T) {
	low := testRule.New("https://b.example/").WithSeverity(SeverityLow)
	high := testRule.New("https://a.example/").WithSeverity(SeverityHigh)
	info := testRule.New("https://a.example/").WithSeverity(SeverityInfo)
	fs := []Finding{low, info, high}
	Sort(fs)
	if fs[0].Severity != SeverityHigh || fs[2].Severity != SeverityInfo {
		t.Errorf("sort order: %v", fs)
	}
	if Max(fs) != SeverityHigh || Max(nil) != 0 {
		t.Error("Max")
	}
	c := Counts(fs)
	if len(c) != 5 || c["high"] != 1 || c["critical"] != 0 {
		t.Errorf("Counts = %v", c)
	}
	if got := AtLeast(fs, SeverityLow); len(got) != 2 {
		t.Errorf("AtLeast = %v", got)
	}
}

func TestRegisterValidatesRules(t *testing.T) {
	bad := []Rule{
		{ID: "BAD", Category: CategoryMetadata, Severity: SeverityLow, Title: "x", Recommendation: "y"},
		{ID: "SEO-X-001", Category: "nope", Severity: SeverityLow, Title: "x", Recommendation: "y"},
		{ID: "SEO-X-002", Category: CategoryMetadata, Severity: SeverityLow, Title: "", Recommendation: "y"},
		{ID: "SEO-X-003", Category: CategoryWebHygiene, Severity: SeverityLow, Title: "x", Recommendation: "y"},
		{ID: "WEB-HYGIENE-999", Category: CategoryMetadata, Severity: SeverityLow, Title: "x", Recommendation: "y"},
	}
	for _, r := range bad {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("rule %q accepted", r.ID)
				}
			}()
			register(r)
		}()
	}
}
