package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

// lineLinks treats every body line starting with "link:" as an href. It
// keeps these tests independent of the HTML parser.
func lineLinks(_ context.Context, p *Page) []*url.URL {
	base, _ := url.Parse(p.Response.FinalURL)
	scope := urlnorm.NewScope(base)
	var out []*url.URL
	for _, line := range strings.Split(string(p.Response.Body), "\n") {
		if ref, ok := strings.CutPrefix(line, "link:"); ok {
			if u, err := urlnorm.Resolve(base, ref); err == nil && scope.Contains(u) {
				out = append(out, u)
			}
		}
	}
	return out
}

type site map[string]string // path -> body ("redirect:/x" redirects)

func (s site) server(t *testing.T, hits *sync.Map) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			v, _ := hits.LoadOrStore(r.URL.RequestURI(), new(atomic.Int32))
			v.(*atomic.Int32).Add(1)
		}
		body, ok := s[r.URL.RequestURI()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if loc, ok := strings.CutPrefix(body, "redirect:"); ok {
			http.Redirect(w, r, loc, http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func startURL(t *testing.T, srv *httptest.Server, p string) *url.URL {
	u, err := urlnorm.Parse(srv.URL + p)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func urls(r *Result) []string {
	var out []string
	for _, p := range r.Pages {
		out = append(out, fmt.Sprintf("%d %s", p.Depth, strings.TrimPrefix(p.URL, "http://127.0.0.1")))
	}
	return out
}

func TestCrawlBreadthFirstAndDeterministic(t *testing.T) {
	s := site{
		"/":       "link:/b\nlink:/a\nlink:https://external.example/\nlink:/a#frag",
		"/a":      "link:/a/deep\nlink:/",
		"/b":      "link:/a/deep\nlink:/missing",
		"/a/deep": "link:/c.pdf",
		"/c.pdf":  "pdf",
	}
	srv := s.server(t, nil)
	var first []string
	for _, conc := range []int{1, 8} {
		r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv, "/"),
			Options{MaxDepth: DefaultMaxDepth, Concurrency: conc, Process: lineLinks})
		got := urls(r)
		port := srv.URL[len("http://127.0.0.1"):]
		want := []string{
			"0 " + port + "/", "1 " + port + "/a", "1 " + port + "/b",
			"2 " + port + "/a/deep", "2 " + port + "/missing", "3 " + port + "/c.pdf",
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("concurrency %d:\n got %v\nwant %v", conc, got, want)
		}
		if first != nil && strings.Join(first, "|") != strings.Join(got, "|") {
			t.Error("result depends on concurrency")
		}
		first = got
		if r.StopReason != StopCompleted {
			t.Errorf("stop reason %q", r.StopReason)
		}
		last := r.Pages[len(r.Pages)-1]
		if !last.Resource || last.Response.Status != 200 {
			t.Errorf("pdf should be a HEAD-checked resource: %+v", last)
		}
		if r.Pages[4].Response.Status != 404 {
			t.Errorf("missing page status %d", r.Pages[4].Response.Status)
		}
	}
}

func TestCrawlLimits(t *testing.T) {
	s := site{"/": ""}
	for i := range 50 {
		s["/"] += fmt.Sprintf("link:/p%02d\n", i)
		s[fmt.Sprintf("/p%02d", i)] = fmt.Sprintf("link:/p%02d/child", i)
	}
	srv := s.server(t, nil)
	f := fetcher.New(fetcher.Options{})

	r := Crawl(context.Background(), f, startURL(t, srv, "/"), Options{MaxDepth: DefaultMaxDepth, MaxPages: 10, Process: lineLinks})
	if len(r.Pages) != 10 || r.StopReason != StopMaxPages || r.SkippedCount["max-pages"] != 50 {
		t.Errorf("max pages: pages=%d stop=%s skipped=%v", len(r.Pages), r.StopReason, r.SkippedCount)
	}
	if !strings.HasSuffix(r.Pages[9].URL, "/p08") {
		t.Errorf("budget not filled in sorted order: last page %s", r.Pages[9].URL)
	}

	r = Crawl(context.Background(), f, startURL(t, srv, "/"), Options{MaxDepth: 1, Process: lineLinks})
	if len(r.Pages) != 51 || r.SkippedCount["max-depth"] != 50 {
		t.Errorf("max depth: pages=%d skipped=%v", len(r.Pages), r.SkippedCount)
	}

	r = Crawl(context.Background(), f, startURL(t, srv, "/"), Options{MaxDepth: 0, Process: lineLinks})
	if len(r.Pages) != 1 {
		t.Errorf("depth 0 crawled %d pages", len(r.Pages))
	}

	r = Crawl(context.Background(), f, startURL(t, srv, "/"), Options{MaxDepth: DefaultMaxDepth, MaxQueue: 5, Process: lineLinks})
	if r.SkippedCount["queue-full"] == 0 {
		t.Errorf("queue limit not applied: %v", r.SkippedCount)
	}
}

func TestCrawlRobotsAndTraps(t *testing.T) {
	s := site{"/": "link:/private/x\nlink:/public"}
	for i := range 40 {
		s["/"] += fmt.Sprintf("\nlink:/search?q=%d", i)
	}
	srv := s.server(t, nil)
	allowed := func(u *url.URL) bool { return !strings.HasPrefix(u.Path, "/private") }
	r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv, "/"),
		Options{MaxDepth: DefaultMaxDepth, Allowed: allowed, Process: lineLinks})
	if r.SkippedCount["robots-disallowed"] != 1 || r.SkippedCount["query-explosion"] != 15 {
		t.Errorf("skipped = %v", r.SkippedCount)
	}
	for _, p := range r.Pages {
		if strings.Contains(p.URL, "/private") {
			t.Error("fetched a disallowed URL")
		}
	}
}

func TestCrawlRedirectTargetsAreNotRefetched(t *testing.T) {
	var hits sync.Map
	s := site{"/": "link:/old\nlink:/new", "/old": "redirect:/new", "/new": "hello"}
	srv := s.server(t, &hits)
	r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv, "/"), Options{MaxDepth: DefaultMaxDepth, Process: lineLinks})
	if len(r.Pages) != 3 {
		t.Fatalf("pages: %v", urls(r))
	}
	s2 := site{"/": "link:/old", "/old": "redirect:/new", "/new": "link:/new"}
	hits = sync.Map{}
	srv2 := s2.server(t, &hits)
	r = Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv2, "/"), Options{MaxDepth: DefaultMaxDepth, Process: lineLinks})
	if n, _ := hits.Load("/new"); n.(*atomic.Int32).Load() != 1 {
		t.Errorf("/new fetched %d times; pages %v", n.(*atomic.Int32).Load(), urls(r))
	}
}

func TestCrawlErrorsAreRecorded(t *testing.T) {
	s := site{"/": "link:/loop", "/loop": "redirect:/loop"}
	srv := s.server(t, nil)
	r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv, "/"), Options{MaxDepth: DefaultMaxDepth, Process: lineLinks})
	if len(r.Pages) != 2 || r.Pages[1].ErrorKind != ErrorRedirectLoop {
		t.Fatalf("pages: %+v", r.Pages)
	}
}

func TestCrawlMaxDurationAndCancel(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write([]byte("link:/next"))
	}))
	defer slow.Close()
	start := startURL(t, slow, "/")

	r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), start, Options{MaxDepth: DefaultMaxDepth, MaxDuration: 100 * time.Millisecond, Process: lineLinks})
	if r.StopReason != StopMaxDuration || r.Pages[0].ErrorKind != ErrorTimeout {
		t.Errorf("stop=%s kind=%s err=%v", r.StopReason, r.Pages[0].ErrorKind, r.Pages[0].Err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	r = Crawl(ctx, fetcher.New(fetcher.Options{}), start, Options{MaxDepth: DefaultMaxDepth, Process: lineLinks})
	if r.StopReason != StopInterrupted || r.Pages[0].ErrorKind != ErrorCanceled {
		t.Errorf("stop=%s kind=%s", r.StopReason, r.Pages[0].ErrorKind)
	}
}

func TestConcurrencyIsBounded(t *testing.T) {
	var cur, peak atomic.Int32
	s := site{"/": ""}
	for i := range 20 {
		s["/"] += fmt.Sprintf("link:/p%d\n", i)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := cur.Add(1)
		defer cur.Add(-1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(s[r.URL.Path]))
	}))
	defer srv.Close()
	Crawl(context.Background(), fetcher.New(fetcher.Options{MaxConnsPerHost: 16}), startURL(t, srv, "/"), Options{MaxDepth: DefaultMaxDepth, Concurrency: 3, Process: lineLinks})
	if peak.Load() > 3 || peak.Load() < 2 {
		t.Errorf("peak concurrency %d, want 2..3", peak.Load())
	}
}

func TestLimiter(t *testing.T) {
	if NewLimiter(0) != nil {
		t.Error("zero rate should disable limiting")
	}
	var nilLimiter *Limiter
	if err := nilLimiter.Wait(context.Background()); err != nil {
		t.Error(err)
	}
	l := NewLimiter(50) // 20ms interval
	start := time.Now()
	for range 6 {
		if err := l.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if el := time.Since(start); el < 90*time.Millisecond {
		t.Errorf("6 requests at 50/s took %v", el)
	}
	slowL := NewLimiter(0.1)
	_ = slowL.Wait(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := slowL.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("cancelled wait: %v", err)
	}
}

func TestClassify(t *testing.T) {
	cases := map[ErrorKind]error{
		"":                    nil,
		ErrorBlocked:          fmt.Errorf("x: %w", &netguard.BlockedError{Class: netguard.ClassLoopback}),
		ErrorRedirectLoop:     fetcher.ErrRedirectLoop,
		ErrorTooManyRedirects: fetcher.ErrTooManyRedirects,
		ErrorTooLarge:         &fetcher.LimitError{Limit: "body"},
		ErrorEncoding:         fetcher.ErrUnsupportedEncoding,
		ErrorInvalidURL:       fetcher.ErrUnsupportedURL,
		ErrorTimeout:          context.DeadlineExceeded,
		ErrorCanceled:         context.Canceled,
		ErrorNetwork:          errors.New("connection refused"),
	}
	for want, err := range cases {
		if got := Classify(err); got != want {
			t.Errorf("Classify(%v) = %q, want %q", err, got, want)
		}
	}
}

func TestIsHTMLAndResources(t *testing.T) {
	if IsHTML(nil) || IsHTML(&fetcher.Response{Status: 404, ContentType: "text/html"}) || IsHTML(&fetcher.Response{Status: 200, ContentType: "image/png"}) {
		t.Error("IsHTML false positives")
	}
	if !IsHTML(&fetcher.Response{Status: 200, ContentType: "application/xhtml+xml"}) {
		t.Error("xhtml not HTML")
	}
	for s, want := range map[string]bool{"/a.PDF": true, "/a.html": false, "/dir/": false, "/img.webp": true} {
		if IsResourceURL(&url.URL{Path: s}) != want {
			t.Errorf("IsResourceURL(%s) != %v", s, want)
		}
	}
}

func TestCrawlStartDisallowed(t *testing.T) {
	var hits sync.Map
	srv := site{"/": "link:/a", "/a": "x"}.server(t, &hits)
	r := Crawl(context.Background(), fetcher.New(fetcher.Options{}), startURL(t, srv, "/"),
		Options{MaxDepth: DefaultMaxDepth, Allowed: func(*url.URL) bool { return false }, Process: lineLinks})
	if len(r.Pages) != 0 || r.StopReason != StopStartDisallowed || r.SkippedCount["robots-disallowed"] != 1 {
		t.Fatalf("pages=%d stop=%s skipped=%v", len(r.Pages), r.StopReason, r.SkippedCount)
	}
	if _, fetched := hits.Load("/"); fetched {
		t.Error("disallowed start URL was fetched")
	}
}
