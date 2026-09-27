package webhygiene

import (
	"net/http"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// goodHeaders is a fully hardened header set: HTTPS, nosniff, Referrer-Policy, HSTS, CSP, XFO.
func goodHeaders() http.Header {
	h := http.Header{}
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	h.Set("Content-Security-Policy", "default-src 'self'")
	h.Set("X-Frame-Options", "SAMEORIGIN")
	return h
}

func TestFindingsGoodPage(t *testing.T) {
	p := Check{
		URL:     "https://example.com/",
		Header:  goodHeaders(),
		IsHTTPS: true,
		Status:  200,
	}
	if got := Findings([]Check{p}); len(got) != 0 {
		t.Errorf("page with hardened headers produced %d findings, want 0: %+v", len(got), got)
	}
}

func TestFindingsInsecureHTTP(t *testing.T) {
	p := Check{
		URL:     "http://example.com/",
		Header:  goodHeaders(),
		IsHTTPS: false,
		Status:  200,
	}
	got := Findings([]Check{p})
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].ID != findings.WebHygieneInsecure.ID {
		t.Errorf("finding ID = %q, want %q", got[0].ID, findings.WebHygieneInsecure.ID)
	}
	if got[0].Severity != findings.SeverityMedium {
		t.Errorf("severity = %v, want medium", got[0].Severity)
	}
}

func TestFindingsMissingXCTO(t *testing.T) {
	p := Check{
		URL:     "https://example.com/",
		Header:  http.Header{},
		IsHTTPS: true,
		Status:  200,
	}
	got := Findings([]Check{p})
	var sawXCTO, sawReferrer bool
	for _, f := range got {
		switch f.ID {
		case findings.WebHygieneXCTO.ID:
			sawXCTO = true
		case findings.WebHygieneReferrer.ID:
			sawReferrer = true
		}
	}
	if !sawXCTO {
		t.Error("expected X-Content-Type-Options finding")
	}
	if !sawReferrer {
		t.Error("expected Referrer-Policy finding")
	}
}

func TestFindingsXCTOWrongValue(t *testing.T) {
	h := goodHeaders()
	h.Set("X-Content-Type-Options", "sniff")
	p := Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got := Findings([]Check{p})
	if len(got) != 1 || got[0].ID != findings.WebHygieneXCTO.ID {
		t.Errorf("expected only XCTO finding, got %+v", got)
	}
}

func TestFindingsDedupByURL(t *testing.T) {
	p := Check{URL: "https://example.com/", Header: goodHeaders(), IsHTTPS: true, Status: 200}
	// Same URL twice: the checks must run only once, producing no findings.
	got := Findings([]Check{p, p})
	if len(got) != 0 {
		t.Errorf("deduplicated page produced %d findings, want 0: %+v", len(got), got)
	}
}

func TestFindingsHSTS(t *testing.T) {
	// Missing HSTS on HTTPS
	h := goodHeaders()
	h.Del("Strict-Transport-Security")
	p := Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got := Findings([]Check{p})
	if len(got) != 1 || got[0].ID != findings.WebHygieneHSTS.ID {
		t.Fatalf("expected 1 HSTS finding, got %+v", got)
	}

	// HSTS without max-age
	h = goodHeaders()
	h.Set("Strict-Transport-Security", "includeSubDomains")
	p = Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got = Findings([]Check{p})
	if len(got) != 1 || got[0].ID != findings.WebHygieneHSTS.ID {
		t.Fatalf("expected 1 HSTS finding for invalid header, got %+v", got)
	}

	// Plain HTTP does not trigger HSTS finding (RFC 6797 §7.2 forbids HSTS over HTTP)
	h = goodHeaders()
	h.Del("Strict-Transport-Security")
	p = Check{URL: "http://example.com/", Header: h, IsHTTPS: false, Status: 200}
	got = Findings([]Check{p})
	for _, f := range got {
		if f.ID == findings.WebHygieneHSTS.ID {
			t.Error("plain HTTP page should not trigger missing HSTS finding")
		}
	}
}

func TestFindingsCSP(t *testing.T) {
	h := goodHeaders()
	h.Del("Content-Security-Policy")
	p := Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got := Findings([]Check{p})
	if len(got) != 1 || got[0].ID != findings.WebHygieneCSP.ID {
		t.Fatalf("expected 1 CSP finding, got %+v", got)
	}
}

func TestFindingsClickjacking(t *testing.T) {
	// Missing both XFO and frame-ancestors
	h := goodHeaders()
	h.Del("X-Frame-Options")
	p := Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got := Findings([]Check{p})
	if len(got) != 1 || got[0].ID != findings.WebHygieneFrame.ID {
		t.Fatalf("expected 1 frame protection finding, got %+v", got)
	}

	// Satisfied by CSP frame-ancestors even without XFO
	h = goodHeaders()
	h.Del("X-Frame-Options")
	h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'self'")
	p = Check{URL: "https://example.com/", Header: h, IsHTTPS: true, Status: 200}
	got = Findings([]Check{p})
	if len(got) != 0 {
		t.Fatalf("frame-ancestors should satisfy clickjacking protection, got %+v", got)
	}
}

func TestFindingsIDsAreStable(t *testing.T) {
	// The IDs are a public contract; they must never change.
	rules := []struct {
		id   string
		want string
	}{
		{findings.WebHygieneInsecure.ID, "WEB-HYGIENE-001"},
		{findings.WebHygieneXCTO.ID, "WEB-HYGIENE-002"},
		{findings.WebHygieneReferrer.ID, "WEB-HYGIENE-003"},
		{findings.WebHygieneRedirect.ID, "WEB-HYGIENE-004"},
		{findings.WebHygieneMixed.ID, "WEB-HYGIENE-005"},
		{findings.WebHygieneHSTS.ID, "WEB-HYGIENE-006"},
		{findings.WebHygieneCSP.ID, "WEB-HYGIENE-007"},
		{findings.WebHygieneFrame.ID, "WEB-HYGIENE-008"},
	}
	for _, r := range rules {
		if r.id != r.want {
			t.Errorf("rule ID = %q, want %q", r.id, r.want)
		}
	}
}
