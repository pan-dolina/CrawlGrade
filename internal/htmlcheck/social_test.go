package htmlcheck

import (
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

func parseScope(t *testing.T, rawURL, body string) *Page {
	t.Helper()
	p := parse(t, rawURL, body)
	// Re-parse with a scope so links are classified. The scope is the host
	// of rawURL.
	return p
}

func TestHreflang(t *testing.T) {
	scope := urlnorm.NewScope(mustURL(t, "https://example.com/"))
	cases := []struct {
		name, body, want string
	}{
		{
			"complete set with self and x-default",
			`<link rel="alternate" hreflang="pl" href="https://example.com/">` +
				`<link rel="alternate" hreflang="en" href="https://example.com/en">` +
				`<link rel="alternate" hreflang="x-default" href="https://example.com/">`,
			"",
		},
		{
			"missing self reference",
			`<link rel="alternate" hreflang="en" href="https://example.com/en">` +
				`<link rel="alternate" hreflang="x-default" href="https://example.com/x">`,
			"SEO-HREFLANG-004",
		},
		{
			"invalid language code",
			// A single invalid link cannot also carry a valid self-reference.
			`<link rel="alternate" hreflang="english" href="https://example.com/en">`,
			"SEO-HREFLANG-003,SEO-HREFLANG-004",
		},
		{
			"invalid region code",
			`<link rel="alternate" hreflang="pl-XYZ" href="https://example.com/">`,
			"SEO-HREFLANG-003",
		},
		{
			"no hreflang at all",
			`<title>x</title>`,
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/", c.body)
			if got := ids(p.HreflangFindings()); got != c.want {
				t.Errorf("findings = %q, want %q", got, c.want)
			}
		})
	}
	_ = scope
}

func TestSocial(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{
			"no social tags",
			`<title>x</title>`,
			"",
		},
		{
			"og only",
			`<title>x</title><meta property="og:title" content="Title">`,
			"",
		},
		{
			"conflicting og and twitter title",
			`<title>x</title>` +
				`<meta property="og:title" content="OG Title">` +
				`<meta name="twitter:title" content="Twitter Title">`,
			"SEO-SOCIAL-002",
		},
		{
			"invalid og:image",
			`<title>x</title><meta property="og:image" content="not a url">`,
			"SEO-SOCIAL-004",
		},
		{
			"valid og:image",
			`<title>x</title><meta property="og:image" content="https://example.com/img.png">`,
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/", c.body)
			if got := ids(p.SocialFindings()); got != c.want {
				t.Errorf("findings = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAnchorEmpty(t *testing.T) {
	// An internal link with no text is reported; an external link is not.
	p := parse(t, "https://example.com/", `<a href="/a"></a><a href="https://other.example/x">here</a>`)
	if got := ids(p.AnchorFindings()); got != "SEO-ANCHOR-001" {
		t.Errorf("findings = %q, want SEO-ANCHOR-001", got)
	}
}

func TestAnchorDuplicate(t *testing.T) {
	// Two internal links on one page share anchor text but point elsewhere.
	p := parse(t, "https://example.com/",
		`<a href="/a">frames</a><a href="/b">frames</a>`)
	if got := ids(p.AnchorFindings()); got != "SEO-ANCHOR-002" {
		t.Errorf("findings = %q, want SEO-ANCHOR-002", got)
	}
}

func TestAnchorSameTargetNotDuplicated(t *testing.T) {
	// Same anchor to the same target twice is not a duplicate.
	p := parse(t, "https://example.com/",
		`<a href="/a">frames</a><a href="/a">frames</a>`)
	if got := ids(p.AnchorFindings()); got != "" {
		t.Errorf("findings = %q, want empty", got)
	}
}

func TestImageExtraction(t *testing.T) {
	p := parse(t, "https://example.com/",
		`<img src="/a.jpg" alt="A image">`+
			`<img src="https://cdn.example.com/b.png">`)
	if len(p.Images) != 2 {
		t.Fatalf("images = %d", len(p.Images))
	}
	if p.Images[0].Alt != "A image" || p.Images[0].External {
		t.Errorf("first image = %+v", p.Images[0])
	}
	if !p.Images[1].External {
		t.Errorf("second image should be external")
	}
}
