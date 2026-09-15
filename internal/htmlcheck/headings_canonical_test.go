package htmlcheck

import (
	"net/url"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

func TestHeadings(t *testing.T) {
	cases := []struct {
		name, body, want string
		opts             CheckOptions
	}{
		{"good", `<h1>Title</h1><h2>A</h2><h3>B</h3><h2>C</h2>`, "", CheckOptions{}},
		{"no h1", `<h2>A</h2>`, "SEO-HEADING-001,SEO-HEADING-004", CheckOptions{}},
		{"multiple", `<h1>A</h1><h1>B</h1>`, "SEO-HEADING-002", CheckOptions{}},
		{"multiple allowed", `<h1>A</h1><h1>B</h1>`, "", CheckOptions{AllowMultipleH1: true}},
		{"empty", `<h1>A</h1><h2> </h2>`, "SEO-HEADING-003", CheckOptions{}},
		{"logo h1", `<h1><img src="logo.png" alt="Salon Optyczny"></h1>`, "", CheckOptions{}},
		{"skip", `<h1>A</h1><h3>B</h3>`, "SEO-HEADING-004", CheckOptions{}},
		{"svg ignored", `<h1>A</h1><svg><h2>x</h2></svg>`, "", CheckOptions{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/", c.body)
			if got := ids(p.HeadingFindings(c.opts)); got != c.want {
				t.Errorf("findings = %q, want %q (headings %+v)", got, c.want, p.Headings)
			}
		})
	}
}

func TestCanonicalParsing(t *testing.T) {
	scope := urlnorm.NewScope(mustURL(t, "https://example.com/"))
	cases := []struct {
		name, body, want, canonical string
	}{
		{"self", `<link rel="canonical" href="https://example.com/page">`, "", "https://example.com/page"},
		{"missing", `<title>x</title>`, "SEO-CANONICAL-001", ""},
		{"relative", `<link rel="Canonical" href="/page">`, "SEO-CANONICAL-003", "https://example.com/page"},
		{"multiple", `<link rel="canonical" href="https://example.com/a"><link rel="canonical" href="https://example.com/b">`, "SEO-CANONICAL-002", "https://example.com/a"},
		{"same twice", `<link rel="canonical" href="https://example.com/a"><link rel="canonical" href="https://EXAMPLE.com/a">`, "", "https://example.com/a"},
		{"javascript", `<link rel="canonical" href="javascript:alert(1)">`, "SEO-CANONICAL-004", ""},
		{"empty", `<link rel="canonical" href="">`, "SEO-CANONICAL-004", ""},
		{"body", `<body><link rel="canonical" href="https://example.com/page"></body>`, "SEO-CANONICAL-005", ""},
		{"cross domain", `<link rel="canonical" href="https://other.example/page">`, "SEO-CANONICAL-006", "https://other.example/page"},
		{"www is same site", `<link rel="canonical" href="https://www.example.com/page">`, "", "https://www.example.com/page"},
		{"rel list", `<link rel="alternate canonical" href="https://example.com/page">`, "", "https://example.com/page"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/page", c.body)
			if got := ids(p.CanonicalFindings(scope)); got != c.want {
				t.Errorf("findings = %q, want %q (%+v)", got, c.want, p.Canonicals)
			}
			if p.CanonicalURL() != c.canonical {
				t.Errorf("canonical = %q, want %q", p.CanonicalURL(), c.canonical)
			}
		})
	}
}

func TestCanonicalHeader(t *testing.T) {
	p := parse(t, "https://example.com/doc.html", `<title>x</title>`)
	p.AddHeaderCanonicals([]string{`<https://example.com/a,b>; rel="preload", <https://example.com/canonical>; rel="canonical"`, `</rel>; rel=canonical`})
	if len(p.Canonicals) != 2 || p.CanonicalURL() != "https://example.com/canonical" || !p.Canonicals[1].Relative {
		t.Fatalf("canonicals = %+v", p.Canonicals)
	}
}

func TestCanonicalSiteFindings(t *testing.T) {
	pages := map[string]*Page{}
	targets := map[string]Target{}
	add := func(u, canonical string, status int, final string) {
		p := parse(t, u, `<link rel="canonical" href="`+canonical+`">`)
		pages[u] = p
		redirects := 0
		if final != "" {
			redirects = 1
		} else {
			final = u
		}
		targets[u] = Target{Status: status, FinalURL: final, Redirects: redirects, Canonical: p.CanonicalURL()}
	}
	add("https://example.com/self", "https://example.com/self", 200, "")
	add("https://example.com/a", "https://example.com/b", 200, "")
	add("https://example.com/b", "https://example.com/a", 200, "")
	add("https://example.com/c", "https://example.com/d", 200, "")
	add("https://example.com/d", "https://example.com/e", 200, "")
	add("https://example.com/e", "https://example.com/e", 200, "")
	add("https://example.com/to-404", "https://example.com/gone", 200, "")
	add("https://example.com/to-redirect", "https://example.com/moved", 200, "")
	add("https://example.com/to-unknown", "https://example.com/never-crawled", 200, "")
	targets["https://example.com/gone"] = Target{Status: 404}
	targets["https://example.com/moved"] = Target{Status: 200, Redirects: 1, FinalURL: "https://example.com/e"}

	var list []*Page
	for _, p := range pages {
		list = append(list, p)
	}
	fs := CanonicalSiteFindings(list, func(u string) (Target, bool) { tg, ok := targets[u]; return tg, ok })
	got := map[string]string{}
	for _, f := range fs {
		got[f.URL] += f.ID + " "
	}
	want := map[string]string{
		"https://example.com/a":           "SEO-CANONICAL-009 ",
		"https://example.com/b":           "SEO-CANONICAL-009 ",
		"https://example.com/c":           "SEO-CANONICAL-010 ",
		"https://example.com/to-404":      "SEO-CANONICAL-007 ",
		"https://example.com/to-redirect": "SEO-CANONICAL-008 ",
	}
	for u, w := range want {
		if got[u] != w {
			t.Errorf("%s: %q, want %q", u, got[u], w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected findings: %v", got)
	}
}

func TestDuplicateH1(t *testing.T) {
	a := parse(t, "https://example.com/a", `<h1>Oferta</h1>`)
	b := parse(t, "https://example.com/b", `<h1>OFERTA</h1>`)
	c := parse(t, "https://example.com/c", `<h2>none</h2>`)
	if got := ids(DuplicateH1Findings([]*Page{a, b, c})); got != "SEO-HEADING-005,SEO-HEADING-005" {
		t.Errorf("findings = %s", got)
	}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
