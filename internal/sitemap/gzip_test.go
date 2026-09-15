package sitemap

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

func gz(t testing.TB, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// bombSitemap is a valid sitemap header followed by n bytes of whitespace,
// which compresses extremely well.
func bombSitemap(t testing.TB, n int) []byte {
	var b bytes.Buffer
	b.WriteString(urlsetHeader)
	b.WriteString("<url><loc>https://example.com/a</loc></url>")
	b.Write(bytes.Repeat([]byte(" "), n))
	b.WriteString("</urlset>")
	return gz(t, b.Bytes())
}

func serveGz(t *testing.T, h http.HandlerFunc) (*httptest.Server, urlnorm.Scope) {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return srv, urlnorm.NewScope(u)
}

func TestGzipSitemapFile(t *testing.T) {
	var body []byte
	srv, scope := serveGz(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		_, _ = w.Write(body)
	})
	body = gz(t, []byte(urlsetHeader+fmt.Sprintf("<url><loc>%s/gz-page</loc></url></urlset>", srv.URL)))
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/sitemap.xml.gz", Declared: true}}, scope, Limits{})
	if len(res.URLs) != 1 || !res.Files[0].Gzip || res.Files[0].Error != "" {
		t.Fatalf("files=%+v urls=%v", res.Files[0], res.URLs)
	}
}

func TestGzipSitemapBombIsRejected(t *testing.T) {
	// 16 MiB of output from roughly 16 KiB of input.
	bomb := bombSitemap(t, 16<<20)
	srv, scope := serveGz(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(bomb)
	})
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/sitemap.xml.gz", Declared: true}}, scope, Limits{MaxBytes: 1 << 20})
	f := res.Files[0]
	if !strings.Contains(f.Error, "decompressed size limit") || len(res.URLs) != 0 {
		t.Fatalf("file = %+v", f)
	}
	if got := ids(res.Findings()); got != "SEO-SITEMAP-002" {
		t.Errorf("findings = %s", got)
	}
}

func TestContentEncodingBombIsRejectedByFetcher(t *testing.T) {
	bomb := bombSitemap(t, 16<<20)
	srv, scope := serveGz(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(bomb)
	})
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/sitemap.xml", Declared: true}}, scope, Limits{MaxBytes: 1 << 20})
	if !strings.Contains(res.Files[0].Error, "decompressed size limit") {
		t.Fatalf("file = %+v", res.Files[0])
	}
}

func TestDoubleGzipIsNotUnpackedTwice(t *testing.T) {
	inner := gz(t, []byte(urlsetHeader+"<url><loc>https://example.com/</loc></url></urlset>"))
	outer := gz(t, inner)
	srv, scope := serveGz(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(outer)
	})
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/s.xml.gz", Declared: true}}, scope, Limits{})
	if res.Files[0].Error == "" {
		t.Fatal("nested gzip accepted")
	}
}

func TestInvalidGzip(t *testing.T) {
	srv, scope := serveGz(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write([]byte("not gzip at all"))
	})
	res := Collect(context.Background(), fetcher.New(fetcher.Options{}), []Seed{{URL: srv.URL + "/s.xml.gz", Declared: true}}, scope, Limits{})
	if !strings.Contains(res.Files[0].Error, "invalid gzip") {
		t.Fatalf("file = %+v", res.Files[0])
	}
}

func TestLimitedBodyBoundary(t *testing.T) {
	for _, n := range []int{0, 1, 99, 100} {
		l := &limitedBody{r: bytes.NewReader(bytes.Repeat([]byte("a"), n)), remain: 100}
		data, err := io.ReadAll(l)
		if err != nil || len(data) != n || l.exceeded {
			t.Errorf("n=%d: len=%d err=%v exceeded=%v", n, len(data), err, l.exceeded)
		}
	}
	l := &limitedBody{r: bytes.NewReader(bytes.Repeat([]byte("a"), 101)), remain: 100}
	if _, err := io.ReadAll(l); !errors.Is(err, fetcher.ErrTooLarge) || !l.exceeded {
		t.Errorf("101 bytes: err=%v", err)
	}
}

func TestIsGzipFile(t *testing.T) {
	cases := []struct {
		resp fetcher.Response
		want bool
	}{
		{fetcher.Response{Body: []byte{0x1f, 0x8b, 0}}, true},
		{fetcher.Response{ContentType: "application/gzip", Body: []byte("x")}, true},
		{fetcher.Response{FinalURL: "https://e.com/s.xml.gz", Body: []byte("\x00\x01")}, true},
		{fetcher.Response{FinalURL: "https://e.com/s.xml.gz", Body: []byte(" <urlset/>")}, false},
		{fetcher.Response{FinalURL: "https://e.com/s.xml", ContentType: "text/xml", Body: []byte("<urlset/>")}, false},
	}
	for i, c := range cases {
		if got := isGzipFile(&c.resp); got != c.want {
			t.Errorf("case %d: %v", i, got)
		}
	}
}
