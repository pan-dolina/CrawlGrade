package report

import (
	"bytes"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"strings"
	"testing"
)

func TestTerminalRejectsControlSequences(t *testing.T) {
	f := findings.TitleMissing.New("https://example.com/", "hostile \x1b[2J\x07 text")
	r := New("https://example.com/", nil, map[string][]findings.Finding{GroupMetadata: {f}})
	var b bytes.Buffer
	if err := RenderTerminal(&b, r); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(b.String(), "\x1b\x07") {
		t.Fatal("terminal controls survived")
	}
}

func TestAggregateDiff(t *testing.T) {
	a := New("https://example.com/", nil, map[string][]findings.Finding{})
	b := New("https://example.com/", nil, map[string][]findings.Finding{})
	a.Summary.Pages = 3
	b.Summary.Pages = 1
	a.Scores = &Scores{SEO: 90}
	b.Scores = &Scores{SEO: 80}
	d, err := Compare(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if d.Equal() || d.PageDelta != 2 || d.SEOScoreDelta != 10 {
		t.Fatal(d)
	}
}
