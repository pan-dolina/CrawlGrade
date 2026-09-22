package audit

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
	"github.com/pan-dolina/crawlgrade/internal/report"
)

// testSite is a small site served by an httptest server. Keys are paths,
// values are bodies. A body beginning with "link:" is treated as a link to
// follow; a body beginning with "redirect:" redirects to the rest.
type testSite map[string]string

func (s testSite) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := s[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if loc, ok := splitPrefix(body, "redirect:"); ok {
			http.Redirect(w, r, loc, http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func splitPrefix(s, p string) (string, bool) {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):], true
	}
	return "", false
}

func testURL(t *testing.T, srv *httptest.Server, path string) *url.URL {
	t.Helper()
	u, err := url.Parse(srv.URL + path)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return u
}

// client returns a fetcher that routes to the test server. AllowPrivate is
// true so the netguard permits the loopback address the test server binds.
func client() *fetcher.Client {
	d := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: true}}
	return fetcher.New(fetcher.Options{DialContext: d.DialContext})
}

func TestAuditCrawlsAndGroupsFindings(t *testing.T) {
	site := testSite{
		// The home page links to /a and /b, which are near-duplicates.
		"/":  "<html><head><title>Home</title></head><body><a href=\"/a\">A</a> <a href=\"/b\">B</a></body></html>",
		"/a": "<html><head><title>Same title</title></head><body>alpha content here</body></html>",
		"/b": "<html><head><title>Same title</title></head><body>alpha content here</body></html>",
	}
	srv := site.server(t)
	start := testURL(t, srv, "/")

	res := Run(context.Background(), client(), start, Options{MaxDepth: 3})
	rep := res.Report
	if rep == nil {
		t.Fatal("audit returned a nil report")
	}
	if rep.Version != "1" {
		t.Errorf("schema_version = %q, want 1", rep.Version)
	}
	if rep.Summary.StartURL != start.String() {
		t.Errorf("start URL = %q, want %q", rep.Summary.StartURL, start.String())
	}
	// The crawl fetched at least the home page and the two linked pages.
	if rep.Summary.Pages < 3 {
		t.Errorf("pages = %d, want at least 3", rep.Summary.Pages)
	}
	// Duplicate titles across /a and /b are metadata findings.
	if rep.FindingCount("metadata") == 0 {
		t.Error("expected metadata findings for duplicate titles")
	}
}

func TestAuditEmptySite(t *testing.T) {
	site := testSite{"/": "<html><head><title>Home</title></head><body>ok</body></html>"}
	srv := site.server(t)
	start := testURL(t, srv, "/")

	res := Run(context.Background(), client(), start, Options{MaxDepth: 3})
	rep := res.Report
	if rep == nil {
		t.Fatal("audit returned a nil report")
	}
	// The page is served over http, so exactly one web hygiene finding is
	// expected: the plain-HTTP observation.
	got := rep.Findings("web-hygiene")
	if len(got) != 1 {
		t.Errorf("web-hygiene findings = %d, want 1 (plain-HTTP)", len(got))
	}
	if len(got) == 1 && got[0].ID != "WEB-HYGIENE-001" {
		t.Errorf("web-hygiene finding = %q, want WEB-HYGIENE-001", got[0].ID)
	}
}

func TestAuditReportRenders(t *testing.T) {
	site := testSite{"/": "link:/a", "/a": "<html><head><title>A</title></head><body>body</body></html>"}
	srv := site.server(t)
	start := testURL(t, srv, "/")

	res := Run(context.Background(), client(), start, Options{MaxDepth: 3})
	rep := res.Report

	// Every format must render without error.
	var buf bytes.Buffer
	if err := rep.Render(&buf, report.FormatTerminal); err != nil {
		t.Errorf("terminal render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("terminal output was empty")
	}
	buf.Reset()
	if err := rep.Render(&buf, report.FormatJSON); err != nil {
		t.Errorf("json render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("json output was empty")
	}
	buf.Reset()
	if err := rep.Render(&buf, report.FormatHTML); err != nil {
		t.Errorf("html render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("html output was empty")
	}
}

func TestUnavailableRobotsDisallowsCrawl(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<a href="/other">other</a>`))
	}))
	defer srv.Close()
	res := Run(context.Background(), client(), testURL(t, srv, "/"), Options{AllowPrivate: true})
	mu.Lock()
	defer mu.Unlock()
	if hits["/"] != 0 || hits["/other"] != 0 {
		t.Fatalf("pages fetched although robots.txt returned 503: %v", hits)
	}
	if res.Report == nil {
		t.Fatal("no report")
	}
}
