package htmlcheck

import (
	"testing"
)

func TestRobotsMeta(t *testing.T) {
	cases := []struct {
		name, body string
		headers    []string
		start      bool
		want       string
		noindex    bool
	}{
		{"none", `<title>x</title>`, nil, false, "", false},
		{"index follow", `<meta name="robots" content="index, follow">`, nil, false, "", false},
		{"noindex", `<meta name="robots" content="noindex">`, nil, false, "SEO-INDEX-001", true},
		{"noindex start", `<meta name="ROBOTS" content="NOINDEX">`, nil, true, "SEO-INDEX-006", true},
		{"none directive", `<meta name="robots" content="none">`, nil, false, "SEO-INDEX-001,SEO-INDEX-002", true},
		{"googlebot noindex", `<meta name="googlebot" content="noindex">`, nil, false, "SEO-INDEX-001", true},
		{"other bot ignored", `<meta name="bingbot" content="noindex">`, nil, false, "", false},
		{"conflict", `<meta name="robots" content="index"><meta name="robots" content="noindex">`, nil, false, "SEO-INDEX-001,SEO-INDEX-003", true},
		{"follow conflict", `<meta name="robots" content="follow, nofollow">`, nil, false, "SEO-INDEX-002,SEO-INDEX-003", false},
		{"invalid", `<meta name="robots" content="max-snippet:abc, max-image-preview:huge, noodp, bogus, max-video-preview">`, nil, false, "SEO-INDEX-004", false},
		{"snippet restricted", `<meta name="robots" content="nosnippet, noarchive, max-image-preview:none">`, nil, false, "SEO-INDEX-005", false},
		{"header noindex", `<title>x</title>`, []string{"noindex, nofollow"}, false, "SEO-INDEX-001,SEO-INDEX-002", true},
		{"header for googlebot", `<title>x</title>`, []string{"googlebot: noindex"}, false, "SEO-INDEX-001", true},
		{"header for other bot", `<title>x</title>`, []string{"otherbot: noindex"}, false, "", false},
		{"header unavailable_after", `<title>x</title>`, []string{"unavailable_after: 25 Jun 2030 15:00:00 PST"}, false, "", false},
		{"header max-snippet not an agent", `<title>x</title>`, []string{"max-snippet: 0"}, false, "SEO-INDEX-005", false},
		{"noindex with canonical elsewhere", `<meta name="robots" content="noindex"><link rel="canonical" href="https://example.com/other">`, nil, false, "SEO-INDEX-001,SEO-INDEX-007", true},
		{"body meta still applies", `<body><meta name="robots" content="noindex"></body>`, nil, false, "SEO-INDEX-001", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/page", c.body)
			p.AddRobotsHeaders(c.headers)
			if got := ids(p.RobotsFindings(c.start)); got != c.want {
				t.Errorf("findings = %q, want %q (%+v)", got, c.want, p.Robots)
			}
			if p.Robots.Indexable() == c.noindex {
				t.Errorf("indexable = %v", p.Robots.Indexable())
			}
		})
	}
}

func TestRobotsMostRestrictiveValues(t *testing.T) {
	p := parse(t, "https://example.com/", `<meta name="robots" content="max-snippet:-1, max-image-preview:large, max-video-preview:30">
<meta name="googlebot" content="max-snippet:50, max-image-preview:standard, max-video-preview:-1, unavailable_after: 2030-01-01">`)
	r := p.Robots
	if r.MaxSnippet == nil || *r.MaxSnippet != 50 || r.MaxImagePreview != "standard" || r.MaxVideoPreview == nil || *r.MaxVideoPreview != 30 || r.UnavailableAfter != "2030-01-01" {
		t.Errorf("robots = %+v snippet=%v video=%v", r, deref(r.MaxSnippet), deref(r.MaxVideoPreview))
	}
	if len(r.Sources) != 2 || r.Sources[1].Agent != "googlebot" {
		t.Errorf("sources = %+v", r.Sources)
	}
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
