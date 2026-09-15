// Package webhygiene implements the passive web hygiene checks. The checks
// read only response headers and markup that CrawlGrade already fetched for
// the SEO audit, plus one extra request to the http:// variant of the start
// URL to test the HTTP to HTTPS redirect. They describe security-relevant
// properties visible to any visitor; they are not a vulnerability scan.
//
// The checks are passive by construction: no payload, probe or unusual
// request is ever sent. A good result does not mean the site is secure.
package webhygiene

import (
	"net/http"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Check is one passive web hygiene observation about a page or the site. It
// carries the response the check reads, so the checks never trigger a request
// of their own.
type Check struct {
	// URL is the page the check applies to, or the start URL for site-wide
	// checks.
	URL string
	// Header is the final response header.
	Header http.Header
	// Body is the decoded response body, if any.
	Body string
	// Redirects lists the redirect hops the request took, if any.
	Redirects []Redirect
	// IsHTTPS reports whether the request used the https scheme.
	IsHTTPS bool
	// Status is the final HTTP status code.
	Status int
}

// Redirect is one hop of a redirect chain.
type Redirect struct {
	URL      string
	Status   int
	Location string
}

// Findings returns the passive web hygiene findings for the checked pages.
// Each page is checked once; the first occurrence of a URL wins.
func Findings(pages []Check) []findings.Finding {
	var out []findings.Finding
	seen := map[string]bool{}
	for _, p := range pages {
		if seen[p.URL] {
			continue
		}
		seen[p.URL] = true
		out = append(out, pageChecks(p)...)
	}
	return out
}

// pageChecks runs the per-page passive checks for one page.
func pageChecks(p Check) []findings.Finding {
	var out []findings.Finding
	if !p.IsHTTPS {
		out = append(out, findings.WebHygieneInsecure.New(p.URL))
	}
	if h := p.Header.Get("X-Content-Type-Options"); h == "" || !strings.EqualFold(h, "nosniff") {
		evidence := "missing"
		if h != "" {
			evidence = "value: " + h
		}
		out = append(out, findings.WebHygieneXCTO.New(p.URL, evidence))
	}
	if p.Header.Get("Referrer-Policy") == "" {
		out = append(out, findings.WebHygieneReferrer.New(p.URL))
	}
	return out
}
