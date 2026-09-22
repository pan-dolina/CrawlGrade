package audit

import (
	"context"
	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/testsite"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fetchFunc func(context.Context, fetcher.Request) (*fetcher.Response, error)

func (f fetchFunc) Fetch(ctx context.Context, r fetcher.Request) (*fetcher.Response, error) {
	return f(ctx, r)
}

func TestRobotsCheckedForWWWOrigin(t *testing.T) {
	var requested []string
	f := fetchFunc(func(_ context.Context, r fetcher.Request) (*fetcher.Response, error) {
		requested = append(requested, r.URL)
		resp := &fetcher.Response{FinalURL: r.URL, Status: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, ContentType: "text/html"}
		switch r.URL {
		case "http://example.com/robots.txt":
			resp.Body = []byte("User-agent: *\nAllow: /")
		case "http://www.example.com/robots.txt":
			resp.Body = []byte("User-agent: *\nDisallow: /")
		case "http://example.com/":
			resp.Body = []byte(`<a href="http://www.example.com/private">other origin</a>`)
		default:
			resp.Status = 404
		}
		return resp, nil
	})
	u, _ := urlnorm.Parse("http://example.com/")
	r := Run(context.Background(), f, u, Options{MaxDepth: 1})
	if len(r.Crawl.Pages) != 1 || !strings.Contains(strings.Join(requested, " "), "www.example.com/robots.txt") {
		t.Fatal(requested)
	}
	for _, u := range requested {
		if strings.HasSuffix(u, "/private") {
			t.Fatal("disallowed origin fetched")
		}
	}
}

func TestAuditDeadlineIncludesDiscovery(t *testing.T) {
	f := fetchFunc(func(ctx context.Context, _ fetcher.Request) (*fetcher.Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	u, _ := urlnorm.Parse("https://example.com/")
	r := Run(context.Background(), f, u, Options{MaxDuration: time.Millisecond})
	if r.Err != context.DeadlineExceeded {
		t.Fatalf("deadline: %v", r.Err)
	}
}

func TestPipelineRetainsAnalysesAfterBodiesAreDropped(t *testing.T) {
	srv := httptest.NewServer(testsite.Handler())
	defer srv.Close()
	f := client()
	defer f.Close()
	r := Run(context.Background(), f, testURL(t, srv, "/"), Options{MaxDepth: 3, MaxPages: 100})
	ids := map[string]bool{}
	for _, fs := range r.Report.Groups {
		for _, f := range fs {
			ids[f.ID] = true
		}
	}
	for _, id := range []string{"SEO-SCHEMA-001", "SEO-IMAGE-001", "SEO-HTTP-001"} {
		if !ids[id] {
			t.Errorf("missing %s", id)
		}
	}
	if len(r.Report.Terms) == 0 {
		t.Fatal("no terms after body disposal")
	}
	for _, p := range r.Crawl.Pages {
		if p.Response != nil && p.Response.Body != nil {
			t.Fatal("body retained")
		}
	}
	orphan := false
	for _, f := range r.Report.Groups["internal-linking"] {
		if f.ID == "SEO-LINK-001" && f.URL == srv.URL+"/orphan" {
			orphan = true
		}
	}
	if !orphan {
		t.Error("sitemap-only orphan not found")
	}
}

func TestHeadersAndZeroDepth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Robots-Tag", "noindex")
		_, _ = w.Write([]byte(`<a href="/child">child</a><main>meaningful content</main>`))
	}))
	defer srv.Close()
	f := client()
	defer f.Close()
	r := Run(context.Background(), f, testURL(t, srv, "/"), Options{MaxDepth: 0})
	if len(r.Crawl.Pages) != 1 || r.Report.Pages[0].Indexable {
		t.Fatal(r.Report.Pages)
	}
	for _, finding := range r.Report.Groups["internal-linking"] {
		if finding.ID == "SEO-LINK-004" {
			t.Fatal("unvisited link reported broken")
		}
	}
}
