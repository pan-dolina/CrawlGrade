package webhygiene

import (
	"net/http"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// goodHeaders is a fully hardened header set: HTTPS, nosniff, Referrer-Policy.
func goodHeaders() http.Header {
	h := http.Header{}
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
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

func TestFindingsIDsAreStable(t *testing.T) {
	// The IDs are a public contract; they must never change.
	if findings.WebHygieneInsecure.ID != "WEB-HYGIENE-001" {
		t.Errorf("insecure ID = %q, want WEB-HYGIENE-001", findings.WebHygieneInsecure.ID)
	}
	if findings.WebHygieneXCTO.ID != "WEB-HYGIENE-002" {
		t.Errorf("XCTO ID = %q, want WEB-HYGIENE-002", findings.WebHygieneXCTO.ID)
	}
	if findings.WebHygieneReferrer.ID != "WEB-HYGIENE-003" {
		t.Errorf("referrer ID = %q, want WEB-HYGIENE-003", findings.WebHygieneReferrer.ID)
	}
}
