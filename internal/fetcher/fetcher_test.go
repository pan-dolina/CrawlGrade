package fetcher

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newClient(opts Options) *Client {
	c := New(opts)
	return c
}

func gzipBytes(t testing.TB, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchPlainAndHeaders(t *testing.T) {
	var ua, ae string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua, ae = r.Header.Get("User-Agent"), r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", "text/HTML; charset=utf-8")
		_, _ = w.Write([]byte("<title>ok</title>"))
	}))
	defer srv.Close()
	c := newClient(Options{UserAgent: "CrawlGrade/test"})
	defer c.Close()
	resp, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 || resp.ContentType != "text/html" || string(resp.Body) != "<title>ok</title>" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if ua != "CrawlGrade/test" || ae != "gzip" {
		t.Errorf("request headers: UA=%q AE=%q", ua, ae)
	}
	if resp.FinalURL != srv.URL+"/" || resp.WireBytes != 17 {
		t.Errorf("final=%q wire=%d", resp.FinalURL, resp.WireBytes)
	}
}

func TestFetchGzip(t *testing.T) {
	payload := []byte(strings.Repeat("hello ", 100))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(gzipBytes(t, payload))
	}))
	defer srv.Close()
	resp, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Body, payload) || resp.ContentEncoding != "gzip" || resp.WireBytes >= int64(len(payload)) {
		t.Errorf("gzip body not decoded correctly: wire=%d body=%d", resp.WireBytes, len(resp.Body))
	}
}

func TestFetchGzipBomb(t *testing.T) {
	// 64 MiB of zeros compresses to about 64 KiB.
	bomb := gzipBytes(t, make([]byte, 64<<20))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(bomb)
	}))
	defer srv.Close()
	c := newClient(Options{MaxBodyBytes: 1 << 20, MaxDecompressedBytes: 2 << 20})
	_, err := c.Fetch(context.Background(), Request{URL: srv.URL})
	var le *LimitError
	if !errors.As(err, &le) || le.Limit != "decompressed" || !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchGiantBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := bytes.Repeat([]byte("a"), 32<<10)
		for i := 0; i < 1024; i++ { // up to 32 MiB, streamed without Content-Length
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	c := newClient(Options{MaxBodyBytes: 256 << 10})
	resp, err := c.Fetch(context.Background(), Request{URL: srv.URL})
	var le *LimitError
	if !errors.As(err, &le) || le.Limit != "body" {
		t.Fatalf("err = %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status should be kept on limit errors, got %d", resp.Status)
	}
}

func TestFetchDeclaredContentLengthTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10485760")
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()
	_, err := newClient(Options{MaxBodyBytes: 1024}).Fetch(context.Background(), Request{URL: srv.URL})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchPerRequestLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 4096))
	}))
	defer srv.Close()
	c := newClient(Options{MaxBodyBytes: 1 << 20})
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL, MaxBodyBytes: 1024}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("per-request body limit: %v", err)
	}
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL, MaxDecompressedBytes: 1024}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("per-request decoded limit: %v", err)
	}
}

func TestFetchUnsupportedAndInvalidEncoding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", r.URL.Query().Get("enc"))
		_, _ = w.Write([]byte("not compressed"))
	}))
	defer srv.Close()
	c := newClient(Options{})
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/?enc=br"}); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Errorf("br: %v", err)
	}
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/?enc=gzip,%20gzip"}); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Errorf("stacked encodings: %v", err)
	}
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/?enc=gzip"}); err == nil || !strings.Contains(err.Error(), "gzip") {
		t.Errorf("invalid gzip: %v", err)
	}
}

func TestFetchRedirectChain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/b", http.StatusMovedPermanently) })
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "c#frag")
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("done")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL + "/a"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.FinalURL != srv.URL+"/c" || len(resp.Redirects) != 2 || string(resp.Body) != "done" {
		t.Fatalf("final=%s redirects=%+v", resp.FinalURL, resp.Redirects)
	}
	if resp.Redirects[0].Status != 301 || resp.Redirects[1].Location != "c#frag" {
		t.Errorf("hops: %+v", resp.Redirects)
	}
}

func TestFetchRedirectLoop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/y", http.StatusFound) })
	mux.HandleFunc("/y", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/x", http.StatusFound) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL + "/x"})
	if !errors.Is(err, ErrRedirectLoop) {
		t.Fatalf("err = %v", err)
	}
	if len(resp.Redirects) != 2 {
		t.Errorf("redirects = %+v", resp.Redirects)
	}
}

func TestFetchTooManyRedirects(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/r"+string(rune('a'+n.Add(1)%26))+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	_, err := newClient(Options{MaxRedirects: 3}).Fetch(context.Background(), Request{URL: srv.URL})
	if !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("err = %v", err)
	}
	if n.Load() != 4 {
		t.Errorf("server saw %d requests, want 4", n.Load())
	}
}

func TestFetchRejectsUnsafeRedirectSchemesAndCredentials(t *testing.T) {
	for _, loc := range []string{"file:///etc/passwd", "gopher://example.com/", "http://user:pass@example.com/", "javascript:alert(1)"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", loc)
			w.WriteHeader(http.StatusFound)
		}))
		_, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL})
		srv.Close()
		if !errors.Is(err, ErrUnsupportedURL) {
			t.Errorf("%s: err = %v", loc, err)
		}
	}
}

func TestFetchRedirectWithoutLocationIsFinal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()
	resp, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL})
	if err != nil || resp.Status != 301 || len(resp.Redirects) != 0 {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestFetchHead(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.Header().Set("Content-Type", "image/png")
	}))
	defer srv.Close()
	resp, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL, Method: http.MethodHead})
	if err != nil || method != "HEAD" || resp.ContentType != "image/png" || resp.Body != nil {
		t.Fatalf("method=%s resp=%+v err=%v", method, resp, err)
	}
	if _, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL, Method: "POST"}); !errors.Is(err, ErrUnsupportedURL) {
		t.Errorf("POST: %v", err)
	}
}

func TestFetchTimeoutAndCancellation(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	start := time.Now()
	_, err := newClient(Options{Timeout: 100 * time.Millisecond}).Fetch(context.Background(), Request{URL: srv.URL})
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("timeout not enforced: %v after %v", err, time.Since(start))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_, err = newClient(Options{Timeout: time.Minute}).Fetch(ctx, Request{URL: srv.URL})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestFetchWaitHookRunsPerHop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/b", http.StatusFound) })
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	var calls atomic.Int32
	c := newClient(Options{Wait: func(context.Context) error { calls.Add(1); return nil }})
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/a"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Errorf("wait called %d times", calls.Load())
	}
	boom := errors.New("limiter closed")
	c = newClient(Options{Wait: func(context.Context) error { return boom }})
	if _, err := c.Fetch(context.Background(), Request{URL: srv.URL + "/a"}); !errors.Is(err, boom) {
		t.Errorf("wait error not returned: %v", err)
	}
}

func TestFetchLargeHeadersRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Big", strings.Repeat("a", 128<<10))
	}))
	defer srv.Close()
	if _, err := newClient(Options{}).Fetch(context.Background(), Request{URL: srv.URL}); err == nil {
		t.Fatal("oversized headers accepted")
	}
}

func TestParseHTTPURL(t *testing.T) {
	for _, bad := range []string{"ftp://example.com/", "http://", "http://u:p@example.com/", "::::", "/relative"} {
		if _, err := ParseHTTPURL(bad); !errors.Is(err, ErrUnsupportedURL) {
			t.Errorf("%q accepted: %v", bad, err)
		}
	}
	if _, err := ParseHTTPURL("https://example.com/path?q=1"); err != nil {
		t.Error(err)
	}
	if (&LimitError{Limit: "body", Max: 3}).Error() == "" {
		t.Error("empty LimitError message")
	}
}
