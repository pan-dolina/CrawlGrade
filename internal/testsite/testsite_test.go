package testsite

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, srv *httptest.Server, path string) (*http.Response, string) {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		r = zr
	}
	body, _ := io.ReadAll(r)
	return resp, string(body)
}

func TestRequiredEndpoints(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	required := map[string]int{
		"/": 200, "/good": 200, "/missing-title": 200, "/duplicate-title": 200,
		"/noindex": 200, "/redirect": 301, "/redirect-chain": 301, "/redirect-loop": 302,
		"/broken": 404, "/canonical-good": 200, "/canonical-loop": 200, "/hreflang": 200,
		"/breadcrumb-jsonld": 200, "/breadcrumb-invalid": 200, "/duplicate-content-a": 200,
		"/duplicate-content-b": 200, "/keyword-page": 200, "/robots.txt": 200, "/sitemap.xml": 200,
		"/ssrf-redirect": 302, "/xss": 200, "/gzip": 200, "/orphan": 200, "/does-not-exist": 404,
		"/calendar?month=2026-12": 200,
	}
	for path, status := range required {
		resp, body := get(t, srv, path)
		if resp.StatusCode != status {
			t.Errorf("%s: status %d, want %d", path, resp.StatusCode, status)
		}
		if strings.Contains(body, "{{base}}") || strings.Contains(resp.Header.Get("Location"), "{{base}}") {
			t.Errorf("%s: unexpanded placeholder", path)
		}
		if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: missing nosniff", path)
		}
	}
}

func TestFixtureDetails(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	if _, body := get(t, srv, "/missing-title"); strings.Contains(body, "<title>") {
		t.Error("missing-title has a title")
	}
	if _, body := get(t, srv, "/robots.txt"); !strings.Contains(body, "Sitemap: "+srv.URL+"/sitemap.xml") {
		t.Errorf("robots.txt:\n%s", body)
	}
	if resp, body := get(t, srv, "/gzip"); resp.Header.Get("Content-Encoding") != "gzip" || !strings.Contains(body, "Kompresja") {
		t.Error("gzip page")
	}
	if resp, _ := get(t, srv, "/redirect-chain-3"); resp.Header.Get("Location") != srv.URL+"/good" {
		t.Errorf("absolute redirect: %q", resp.Header.Get("Location"))
	}
	_, a := get(t, srv, "/duplicate-content-a")
	_, b := get(t, srv, "/duplicate-content-b")
	_, c := get(t, srv, "/near-duplicate")
	if a == b || a == c || !strings.Contains(c, "Pamiętaj również") {
		t.Error("duplicate fixtures must differ outside main content")
	}
	if _, body := get(t, srv, "/calendar?month=2026-12"); !strings.Contains(body, "month=2027-01") {
		t.Error("calendar does not advance")
	}
	if _, body := get(t, srv, "/calendar?month=bogus"); !strings.Contains(body, "month=2026-02") {
		t.Error("calendar default")
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST status %d", resp.StatusCode)
	}
	head, _ := http.NewRequest(http.MethodHead, srv.URL+"/good", nil)
	resp, err = http.DefaultClient.Do(head)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("HEAD status %d", resp.StatusCode)
	}
	if len(Paths()) < 30 {
		t.Errorf("only %d paths", len(Paths()))
	}
}
