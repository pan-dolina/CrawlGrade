package sitemap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

const urlsetHeader = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">`

func TestParseURLSet(t *testing.T) {
	doc, err := Parse(strings.NewReader(urlsetHeader + `
  <url><loc> https://example.com/a </loc><lastmod>2024-05-01</lastmod><image:image><image:loc>https://example.com/i.png</image:loc></image:image></url>
  <url><loc>https://example.com/b?x=1&amp;y=2</loc><lastmod>yesterday</lastmod><changefreq>daily</changefreq></url>
  <url><lastmod>2024-05-01T10:00:00+02:00</lastmod></url>
  <url><loc>https://example.com/` + strings.Repeat("x", MaxLocLength) + `</loc></url>
  <unknown><nested><deep/></nested></unknown>
</urlset>`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Kind != KindURLSet || len(doc.Entries) != 2 {
		t.Fatalf("doc = %+v", doc)
	}
	if doc.Entries[0].Loc != "https://example.com/a" || doc.Entries[1].Loc != "https://example.com/b?x=1&y=2" {
		t.Errorf("entries = %+v", doc.Entries)
	}
	want := []string{`invalid <lastmod> "yesterday"`, "<url> without <loc>", "<loc> longer than"}
	for i, w := range want {
		if i >= len(doc.Warnings) || !strings.Contains(doc.Warnings[i], w) {
			t.Errorf("warnings = %q", doc.Warnings)
			break
		}
	}
}

func TestParseIndexAndErrors(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>https://example.com/s1.xml</loc></sitemap></sitemapindex>`))
	if err != nil || doc.Kind != KindSitemapIndex || len(doc.Entries) != 1 || len(doc.Warnings) != 0 {
		t.Fatalf("doc=%+v err=%v", doc, err)
	}
	doc, err = Parse(strings.NewReader(`<urlset><url><loc>https://example.com/</loc></url></urlset>`))
	if err != nil || !strings.Contains(doc.Warnings[0], "namespace") {
		t.Errorf("missing namespace warning: %+v %v", doc, err)
	}
	for name, input := range map[string]string{
		"html":        `<html><body>Not found</body></html>`,
		"empty":       ``,
		"truncated":   urlsetHeader + `<url><loc>https://example.com/`,
		"mismatched":  urlsetHeader + `<url><loc>x</lastmod></url></urlset>`,
		"latin1":      `<?xml version="1.0" encoding="ISO-8859-2"?><urlset/>`,
		"entity":      `<!DOCTYPE urlset [<!ENTITY a "aaaaaaaaaa">]><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>&a;</loc></url></urlset>`,
		"external":    `<!DOCTYPE urlset [<!ENTITY x SYSTEM "file:///etc/passwd">]><urlset><url><loc>&x;</loc></url></urlset>`,
		"not closing": `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
	} {
		doc, err := Parse(strings.NewReader(input))
		if name == "not closing" {
			// Truncated after the root element: reported, entries so far kept.
			if err == nil {
				t.Errorf("%s: no error", name)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: accepted: %+v", name, doc)
		}
	}
	if _, err := Parse(strings.NewReader(`<feed/>`)); !errors.Is(err, ErrNotSitemap) {
		t.Errorf("feed: %v", err)
	}
}

func TestParseEntryLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString(urlsetHeader)
	for i := range MaxEntriesPerFile + 5 {
		fmt.Fprintf(&b, "<url><loc>https://example.com/%d</loc></url>", i)
	}
	b.WriteString("</urlset>")
	doc, err := Parse(strings.NewReader(b.String()))
	if err != nil || len(doc.Entries) != MaxEntriesPerFile || !doc.Truncated {
		t.Fatalf("entries=%d truncated=%v err=%v", len(doc.Entries), doc.Truncated, err)
	}
}

func TestValidLastMod(t *testing.T) {
	for _, s := range []string{"2024", "2024-05", "2024-05-01", "2024-05-01T10:00+02:00", "2024-05-01T10:00:00Z", "2024-05-01T10:00:00.123Z"} {
		if !ValidLastMod(s) {
			t.Errorf("%s rejected", s)
		}
	}
	for _, s := range []string{"01/05/2024", "2024-13-01", "2024-05-01 10:00", "now"} {
		if ValidLastMod(s) {
			t.Errorf("%s accepted", s)
		}
	}
}

type testSite struct {
	files map[string]string
	hits  atomic.Int32
}

func (s *testSite) serve(t *testing.T) (*httptest.Server, urlnorm.Scope) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		body, ok := s.files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(strings.ReplaceAll(body, "{{base}}", "http://"+r.Host)))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return srv, urlnorm.NewScope(u)
}

func ids(fs []findings.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.ID)
	}
	return strings.Join(s, ",")
}

func TestCollectIndexAndURLs(t *testing.T) {
	site := &testSite{files: map[string]string{
		"/index.xml": `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<sitemap><loc>{{base}}/pages.xml</loc></sitemap>
			<sitemap><loc>{{base}}/posts.xml</loc></sitemap>
			<sitemap><loc>{{base}}/missing.xml</loc></sitemap>
			<sitemap><loc>{{base}}/pages.xml</loc></sitemap>
			<sitemap><loc>{{base}}/nested.xml</loc></sitemap>
		</sitemapindex>`,
		"/pages.xml": urlsetHeader + `<url><loc>{{base}}/b</loc></url><url><loc>{{base}}/a</loc></url><url><loc>https://other.example/x</loc></url><url><loc>javascript:alert(1)</loc></url></urlset>`,
		"/posts.xml": urlsetHeader + `<url><loc>{{base}}/a</loc></url><url><loc>{{base}}/c#frag</loc></url></urlset>`,
		"/nested.xml": `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<sitemap><loc>{{base}}/deeper.xml</loc></sitemap></sitemapindex>`,
		"/deeper.xml": `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>{{base}}/never.xml</loc></sitemap></sitemapindex>`,
	}}
	srv, scope := site.serve(t)
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/index.xml", Source: "robots.txt", Declared: true}}, scope, Limits{})

	var locs []string
	for _, u := range res.URLs {
		locs = append(locs, strings.TrimPrefix(u.Loc, srv.URL))
	}
	if strings.Join(locs, ",") != "/a,/b,/c" {
		t.Errorf("URLs = %v", locs)
	}
	if !res.Contains(srv.URL+"/b") || res.Contains(srv.URL+"/zzz") {
		t.Error("Contains")
	}
	if len(res.OutOfScope) != 1 || len(res.InvalidLocs) != 1 {
		t.Errorf("out of scope=%v invalid=%v", res.OutOfScope, res.InvalidLocs)
	}
	if len(res.Files) != 6 {
		for _, f := range res.Files {
			t.Logf("%+v", f)
		}
		t.Fatalf("files = %d", len(res.Files))
	}
	got := ids(res.Findings())
	want := "SEO-SITEMAP-002,SEO-SITEMAP-005,SEO-SITEMAP-005,SEO-SITEMAP-006,SEO-SITEMAP-007"
	if got != want {
		t.Errorf("findings = %s, want %s", got, want)
	}
}

func TestCollectMissingAndLimits(t *testing.T) {
	site := &testSite{files: map[string]string{}}
	srv, scope := site.serve(t)
	u, _ := url.Parse(srv.URL + "/start")
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{DefaultSeed(u)}, scope, Limits{})
	if got := ids(res.Findings()); got != "SEO-SITEMAP-001" {
		t.Errorf("missing sitemap findings = %s", got)
	}

	var idx strings.Builder
	idx.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	site.files = map[string]string{}
	for i := range 10 {
		fmt.Fprintf(&idx, "<sitemap><loc>{{base}}/s%d.xml</loc></sitemap>", i)
		site.files[fmt.Sprintf("/s%d.xml", i)] = urlsetHeader + fmt.Sprintf("<url><loc>{{base}}/p%d-1</loc></url><url><loc>{{base}}/p%d-2</loc></url></urlset>", i, i)
	}
	idx.WriteString("<sitemap><loc>not a url</loc></sitemap></sitemapindex>")
	site.files["/sitemap.xml"] = idx.String()
	site.hits.Store(0)
	res = Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{DefaultSeed(u)}, scope, Limits{MaxFiles: 4, MaxURLs: 5})
	if site.hits.Load() != 4 || res.FilesSkipped != 8 || len(res.URLs) != 5 || res.URLsSkipped != 1 {
		t.Errorf("hits=%d filesSkipped=%d urls=%d urlsSkipped=%d", site.hits.Load(), res.FilesSkipped, len(res.URLs), res.URLsSkipped)
	}
	if got := ids(res.Findings()); got != "SEO-SITEMAP-008" {
		t.Errorf("limit findings = %s", got)
	}
}

func TestCollectInvalidDocuments(t *testing.T) {
	site := &testSite{files: map[string]string{
		"/sitemap.xml": "<html>soft 404</html>",
	}}
	srv, scope := site.serve(t)
	u, _ := url.Parse(srv.URL)
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{DefaultSeed(u), {URL: "::bad::", Source: "robots.txt", Declared: true}}, scope, Limits{})
	if got := ids(res.Findings()); got != "SEO-SITEMAP-002,SEO-SITEMAP-003" {
		t.Errorf("findings = %s", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	site.hits.Store(0)
	Collect(ctx, fetcher.New(fetcher.Options{}), []Seed{DefaultSeed(u)}, scope, Limits{})
	if site.hits.Load() != 0 {
		t.Error("cancelled collection made requests")
	}
}
