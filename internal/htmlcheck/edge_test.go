package htmlcheck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

func TestTitleEdgeCases(t *testing.T) {
	cases := map[string]string{
		`<title>Archiwum &amp; okładki &ndash; salon</title>`:        "Archiwum & okładki – salon",
		`<title>Tytuł z <b>tagiem</b> w środku</title>`:              "Tytuł z <b>tagiem</b> w środku",
		"<title>\n\t Nowa\nlinia \r\n</title>":                       "Nowa linia",
		`<title>&nbsp;&#8203;</title>`:                               "\xe2\x80\x8b",
		`<title>&lt;script&gt;alert(1)&lt;/script&gt;</title>`:       "<script>alert(1)</script>",
		`<head><title>First</title></head><body><title>Body</title>`: "First",
	}
	for body, want := range cases {
		p := parse(t, "https://example.com/", body)
		if p.Title() != want {
			t.Errorf("%q: title = %q, want %q", body, p.Title(), want)
		}
	}
	// Only whitespace entities: the title is empty.
	p := parse(t, "https://example.com/", `<title>&nbsp; &#x2003;</title>`)
	if ids(p.MetadataFindings()) != "SEO-TITLE-002,SEO-DESC-001" {
		t.Errorf("whitespace-only title: %s", ids(p.MetadataFindings()))
	}
	// A second <title> in the body is ignored when the head has one.
	p = parse(t, "https://example.com/", `<head><title>Salon archiwalny w regionie</title></head><body><title>Body</title></body>`)
	if len(p.Titles) != 1 {
		t.Errorf("titles = %q", p.Titles)
	}
}

func TestDescriptionEdgeCases(t *testing.T) {
	cases := []struct {
		body string
		want []string
	}{
		{`<meta name="DESCRIPTION" content="Upper">`, []string{"Upper"}},
		{`<meta name=" description " content="Spaced name">`, []string{"Spaced name"}},
		{"<meta name=description content='Line\nbreak\tand  tabs'>", []string{"Line break and tabs"}},
		{`<meta name="description">`, []string{""}},
		{`<meta name="og:description" content="not it"><meta itemprop="description" content="nor this">`, nil},
		{`<meta content="reversed order" name="description">`, []string{"reversed order"}},
	}
	for _, c := range cases {
		p := parse(t, "https://example.com/", c.body)
		if fmt.Sprint(p.Descriptions) != fmt.Sprint(c.want) {
			t.Errorf("%q: descriptions = %q, want %q", c.body, p.Descriptions, c.want)
		}
	}
}

func TestDuplicateMetadataIsExactAfterCollapsing(t *testing.T) {
	a := parse(t, "https://example.com/a", `<title>Oferta</title>`)
	b := parse(t, "https://example.com/b", `<title>oferta</title>`)
	c := parse(t, "https://example.com/c", "<title> Oferta\n</title>")
	fs := DuplicateMetadataFindings([]*Page{a, b, c})
	if got := ids(fs); got != "SEO-TITLE-004,SEO-TITLE-004" {
		t.Fatalf("findings = %s", got)
	}
	for _, f := range fs {
		if f.URL == "https://example.com/b" {
			t.Error("titles differing in case are not duplicates")
		}
	}
	// Three identical titles: each page lists the two others.
	d := parse(t, "https://example.com/d", `<title>Oferta</title>`)
	fs = DuplicateMetadataFindings([]*Page{a, c, d})
	if len(fs) != 3 || len(fs[0].Evidence) != 3 {
		t.Errorf("findings = %+v", fs)
	}
}

func TestCanonicalEdgeCases(t *testing.T) {
	scope := urlnorm.NewScope(mustURL(t, "https://example.com/"))
	cases := []struct {
		name, page, body, canonical, want string
	}{
		{"whitespace in href", "https://example.com/p", "<link rel=canonical href=\"\n https://example.com/p \t\">", "https://example.com/p", ""},
		{"fragment dropped", "https://example.com/p", `<link rel="canonical" href="https://example.com/p#top">`, "https://example.com/p", ""},
		{"host case and default port", "https://example.com/p", `<link rel="canonical" href="HTTPS://EXAMPLE.COM:443/p">`, "https://example.com/p", ""},
		{"http page https canonical", "http://example.com/p", `<link rel="canonical" href="https://example.com/p">`, "https://example.com/p", ""},
		{"credentials", "https://example.com/p", `<link rel="canonical" href="https://user:pw@example.com/p">`, "", "SEO-CANONICAL-004"},
		{"protocol relative", "https://example.com/p", `<link rel="canonical" href="//example.com/p">`, "https://example.com/p", "SEO-CANONICAL-003"},
		{"noscript is inert", "https://example.com/p", `<head><noscript><link rel="canonical" href="https://example.com/other"></noscript></head>`, "", "SEO-CANONICAL-001"},
		{"idn canonical", "https://example.com/p", `<link rel="canonical" href="https://bücher.example/p">`, "https://xn--bcher-kva.example/p", "SEO-CANONICAL-006"},
		{"valid and invalid", "https://example.com/p", `<link rel="canonical" href="https://example.com/p"><link rel="canonical" href="ftp://example.com/p">`, "https://example.com/p", "SEO-CANONICAL-002,SEO-CANONICAL-004"},
		{"body only plus head", "https://example.com/p", `<head><link rel="canonical" href="https://example.com/p"></head><body><link rel="canonical" href="https://example.com/q"></body>`, "https://example.com/p", "SEO-CANONICAL-002,SEO-CANONICAL-005"},
		{"base href relative", "https://example.com/dir/p", `<head><base href="https://example.com/other/"><link rel="canonical" href="p"></head>`, "https://example.com/other/p", "SEO-CANONICAL-003"},
		{"svg link ignored", "https://example.com/p", `<link rel="canonical" href="https://example.com/p"><body><svg><link rel="canonical" href="https://example.com/x"/></svg></body>`, "https://example.com/p", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, c.page, c.body)
			if got := ids(p.CanonicalFindings(scope)); got != c.want {
				t.Errorf("findings = %q, want %q (%+v)", got, c.want, p.Canonicals)
			}
			if p.CanonicalURL() != c.canonical {
				t.Errorf("canonical = %q, want %q", p.CanonicalURL(), c.canonical)
			}
		})
	}
}

func TestCanonicalLimitAndHeaderConflicts(t *testing.T) {
	body := strings.Repeat(`<link rel="canonical" href="https://example.com/p">`, MaxCanonicals+10)
	p := parse(t, "https://example.com/p", body)
	if len(p.Canonicals) != MaxCanonicals {
		t.Errorf("canonicals = %d", len(p.Canonicals))
	}
	p = parse(t, "https://example.com/p", `<link rel="canonical" href="https://example.com/p">`)
	p.AddHeaderCanonicals([]string{`<https://example.com/other>; rel="canonical"`, `garbage`, `<https://example.com/x>`, `<https://example.com/y>; rel="next"`})
	scope := urlnorm.NewScope(mustURL(t, "https://example.com/"))
	if got := ids(p.CanonicalFindings(scope)); got != "SEO-CANONICAL-002" {
		t.Errorf("header conflict: %s (%+v)", got, p.Canonicals)
	}
}

func TestCanonicalSiteEdgeCases(t *testing.T) {
	a := parse(t, "https://example.com/a", `<link rel="canonical" href="https://example.com/err">`)
	b := parse(t, "https://example.com/b", `<link rel="canonical" href="https://example.com/b">`)
	lookup := func(u string) (Target, bool) {
		if u == "https://example.com/err" {
			return Target{Error: "connection refused"}, true
		}
		return Target{}, false
	}
	fs := CanonicalSiteFindings([]*Page{a, b}, lookup)
	if ids(fs) != "SEO-CANONICAL-007" || !strings.Contains(strings.Join(fs[0].Evidence, " "), "connection refused") {
		t.Errorf("findings = %+v", fs)
	}

	// A long chain is reported once and terminates.
	var pages []*Page
	targets := map[string]Target{}
	for i := range 50 {
		u := fmt.Sprintf("https://example.com/%d", i)
		next := fmt.Sprintf("https://example.com/%d", i+1)
		if i == 49 {
			next = u
		}
		p := parse(t, u, `<link rel="canonical" href="`+next+`">`)
		pages = append(pages, p)
		targets[u] = Target{Status: 200, FinalURL: u, Canonical: next}
	}
	fs = CanonicalSiteFindings(pages[:1], func(u string) (Target, bool) { tg, ok := targets[u]; return tg, ok })
	if ids(fs) != "SEO-CANONICAL-010" {
		t.Errorf("chain findings = %s", ids(fs))
	}
}
