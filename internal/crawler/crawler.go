// Package crawler walks a site breadth-first within explicit limits.
//
// The crawl is level-synchronous (ADR 0002): all URLs at one depth are
// fetched concurrently by a bounded worker pool, then the links they
// produced are de-duplicated, filtered, sorted and truncated to the
// remaining budget to form the next level. The set of crawled pages is
// therefore independent of timing and --concurrency.
package crawler

import (
	"context"
	"errors"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
)

// Defaults. See docs/crawling.md.
const (
	DefaultMaxPages          = 500
	DefaultMaxDepth          = 10
	DefaultConcurrency       = 4
	DefaultRequestsPerSecond = 5
	DefaultMaxDuration       = 10 * time.Minute
	MaxConcurrency           = 64
	maxSkippedRecords        = 1000
)

// Fetcher performs requests; *fetcher.Client implements it.
type Fetcher interface {
	Fetch(ctx context.Context, req fetcher.Request) (*fetcher.Response, error)
}

// Options configures a crawl.
type Options struct {
	MaxPages int
	Seeds    []*url.URL
	// MaxDepth is the link distance from the start URL; 0 fetches only the
	// start URL. Unlike the other limits, the zero value is meaningful.
	MaxDepth    int
	Concurrency int
	// MaxQueue bounds the number of URLs waiting in the next level.
	MaxQueue    int
	MaxDuration time.Duration
	TrapLimits  urlnorm.TrapLimits
	// Allowed reports whether robots.txt permits fetching u. Nil allows all.
	Allowed func(u *url.URL) bool
	// Process analyses a fetched HTML page and returns the links to follow.
	// It runs on worker goroutines and must be safe for concurrent use. It
	// may store analysis results in page.Data; the crawler drops the response
	// body afterwards to bound memory.
	Process func(ctx context.Context, page *Page) []*url.URL
}

func (o *Options) setDefaults() {
	if o.MaxPages <= 0 {
		o.MaxPages = DefaultMaxPages
	}
	if o.MaxDepth < 0 {
		o.MaxDepth = 0
	}
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	o.Concurrency = min(o.Concurrency, MaxConcurrency)
	if o.MaxQueue <= 0 {
		o.MaxQueue = min(max(10*o.MaxPages, 1000), 100000)
	}
	if o.MaxDuration <= 0 {
		o.MaxDuration = DefaultMaxDuration
	}
}

// Page is one fetched URL.
type Page struct {
	URL      string // normalized URL that was requested
	Depth    int
	Resource bool // fetched with HEAD because the URL names a non-HTML file
	Response *fetcher.Response
	// ErrorKind classifies Err; empty when the fetch succeeded.
	ErrorKind ErrorKind
	Err       error
	// Links are the in-scope URLs Process returned.
	Links []string
	// Data holds analysis results set by Options.Process.
	Data any
}

// ErrorKind classifies fetch failures for reports.
type ErrorKind string

// Error kinds.
const (
	ErrorBlocked          ErrorKind = "blocked"
	ErrorRedirectLoop     ErrorKind = "redirect-loop"
	ErrorTooManyRedirects ErrorKind = "too-many-redirects"
	ErrorTooLarge         ErrorKind = "too-large"
	ErrorTimeout          ErrorKind = "timeout"
	ErrorEncoding         ErrorKind = "unsupported-encoding"
	ErrorInvalidURL       ErrorKind = "invalid-url"
	ErrorCanceled         ErrorKind = "canceled"
	ErrorNetwork          ErrorKind = "network"
)

// Classify maps a fetch error to an ErrorKind.
func Classify(err error) ErrorKind {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, netguard.ErrBlocked):
		return ErrorBlocked
	case errors.Is(err, fetcher.ErrRedirectLoop):
		return ErrorRedirectLoop
	case errors.Is(err, fetcher.ErrTooManyRedirects):
		return ErrorTooManyRedirects
	case errors.Is(err, fetcher.ErrTooLarge):
		return ErrorTooLarge
	case errors.Is(err, fetcher.ErrUnsupportedEncoding):
		return ErrorEncoding
	case errors.Is(err, fetcher.ErrUnsupportedURL):
		return ErrorInvalidURL
	case errors.Is(err, context.DeadlineExceeded) || isTimeout(err):
		return ErrorTimeout
	case errors.Is(err, context.Canceled):
		return ErrorCanceled
	}
	return ErrorNetwork
}

func isTimeout(err error) bool {
	var t interface{ Timeout() bool }
	return errors.As(err, &t) && t.Timeout()
}

// SkipReason explains why a discovered URL was not fetched.
type SkipReason string

// Skip reasons.
const (
	SkipRobots    SkipReason = "robots-disallowed"
	SkipMaxDepth  SkipReason = "max-depth"
	SkipMaxPages  SkipReason = "max-pages"
	SkipQueueFull SkipReason = "queue-full"
)

// Skipped records a URL that was discovered but not fetched.
type Skipped struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// StopReason explains why the crawl ended.
type StopReason string

// Stop reasons.
const (
	StopCompleted   StopReason = "completed"
	StopMaxPages    StopReason = "max-pages"
	StopMaxDuration StopReason = "max-duration"
	StopInterrupted StopReason = "interrupted"
	// StopStartDisallowed: robots.txt disallows the start URL (or is
	// unavailable, which RFC 9309 treats as disallowing everything).
	StopStartDisallowed StopReason = "start-disallowed"
)

// Result is the outcome of a crawl.
type Result struct {
	Pages        []*Page // in crawl order: by depth, then URL
	Skipped      []Skipped
	SkippedCount map[string]int
	StopReason   StopReason
}

// Crawl fetches start and follows the links Process returns.
func Crawl(ctx context.Context, f Fetcher, start *url.URL, opts Options) *Result {
	opts.setDefaults()
	ctx, cancel := context.WithTimeout(ctx, opts.MaxDuration)
	defer cancel()

	c := &crawl{
		opts:   opts,
		scope:  urlnorm.NewScope(start),
		f:      f,
		guard:  urlnorm.NewTrapGuard(opts.TrapLimits),
		seen:   map[string]bool{},
		result: &Result{SkippedCount: map[string]int{}, StopReason: StopCompleted},
	}
	c.guard.Admit(start)
	c.seen[start.String()] = true
	if opts.Allowed != nil && !opts.Allowed(start) {
		c.skip(start.String(), SkipRobots)
		c.result.StopReason = StopStartDisallowed
		return c.result
	}
	level := []*url.URL{start}

	for depth := 0; len(level) > 0; depth++ {
		remaining := opts.MaxPages - len(c.result.Pages)
		if remaining <= 0 {
			c.skipAll(level, SkipMaxPages)
			c.result.StopReason = StopMaxPages
			break
		}
		if len(level) > remaining {
			c.skipAll(level[remaining:], SkipMaxPages)
			level = level[:remaining]
			c.result.StopReason = StopMaxPages
		}
		pages := c.runLevel(ctx, level, depth)
		c.result.Pages = append(c.result.Pages, pages...)
		if err := ctx.Err(); err != nil {
			c.result.StopReason = stopReason(err)
			break
		}
		level = c.nextLevel(pages, depth+1)
	}
	return c.result
}

func stopReason(err error) StopReason {
	if errors.Is(err, context.DeadlineExceeded) {
		return StopMaxDuration
	}
	return StopInterrupted
}

type crawl struct {
	scope  urlnorm.Scope
	opts   Options
	f      Fetcher
	guard  *urlnorm.TrapGuard
	seen   map[string]bool
	result *Result
}

func (c *crawl) skip(u string, reason SkipReason) {
	c.result.SkippedCount[string(reason)]++
	if len(c.result.Skipped) < maxSkippedRecords {
		c.result.Skipped = append(c.result.Skipped, Skipped{URL: u, Reason: string(reason)})
	}
}

func (c *crawl) skipAll(us []*url.URL, reason SkipReason) {
	for _, u := range us {
		c.skip(u.String(), reason)
	}
}

func (c *crawl) runLevel(ctx context.Context, level []*url.URL, depth int) []*Page {
	pages := make([]*Page, len(level))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(c.opts.Concurrency, len(level)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				pages[i] = c.fetchPage(ctx, level[i], depth)
			}
		}()
	}
	for i := range level {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return pages
}

func (c *crawl) fetchPage(ctx context.Context, u *url.URL, depth int) *Page {
	p := &Page{URL: u.String(), Depth: depth, Resource: IsResourceURL(u)}
	if ctx.Err() != nil {
		p.Err = ctx.Err()
		p.ErrorKind = ErrorCanceled
		return p
	}
	req := fetcher.Request{URL: p.URL}
	if p.Resource {
		req.Method = "HEAD"
	}
	resp, err := c.f.Fetch(ctx, req)
	p.Response = resp
	p.Err = err
	p.ErrorKind = Classify(err)
	if err == nil && !p.Resource && c.opts.Process != nil && IsHTML(resp) {
		for _, l := range c.opts.Process(ctx, p) {
			p.Links = append(p.Links, l.String())
		}
	}
	if resp != nil {
		resp.Body = nil
	}
	return p
}

// nextLevel collects the links of pages, in page order then link order,
// and returns the admissible unseen URLs sorted.
func (c *crawl) nextLevel(pages []*Page, depth int) []*url.URL {
	var candidates []string
	if depth == 1 {
		for _, u := range c.opts.Seeds {
			if u != nil && c.scope.Contains(u) {
				candidates = append(candidates, u.String())
			}
		}
	}
	for _, p := range pages {
		// A redirect target is considered seen so it is not fetched again.
		if p.Response != nil && p.Response.FinalURL != "" {
			if fu, err := urlnorm.Parse(p.Response.FinalURL); err == nil {
				c.seen[fu.String()] = true
			}
		}
		candidates = append(candidates, p.Links...)
	}
	slices.Sort(candidates)
	candidates = slices.Compact(candidates)

	var next []*url.URL
	for _, s := range candidates {
		if c.seen[s] {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			continue
		}
		c.seen[s] = true
		if depth > c.opts.MaxDepth {
			c.skip(s, SkipMaxDepth)
			continue
		}
		if c.opts.Allowed != nil && !c.opts.Allowed(u) {
			c.skip(s, SkipRobots)
			continue
		}
		if reason := c.guard.Admit(u); reason != urlnorm.TrapNone {
			c.skip(s, SkipReason(reason))
			continue
		}
		if len(next) >= c.opts.MaxQueue {
			c.skip(s, SkipQueueFull)
			continue
		}
		next = append(next, u)
	}
	return next
}

// IsHTML reports whether a response carries an HTML document.
func IsHTML(resp *fetcher.Response) bool {
	if resp == nil || resp.Status < 200 || resp.Status > 299 {
		return false
	}
	switch resp.ContentType {
	case "text/html", "application/xhtml+xml", "":
		return true
	}
	return false
}

var resourceExtensions = map[string]bool{
	".7z": true, ".avi": true, ".avif": true, ".bmp": true, ".css": true, ".csv": true,
	".doc": true, ".docx": true, ".eot": true, ".exe": true, ".gif": true, ".gz": true,
	".ico": true, ".jpeg": true, ".jpg": true, ".js": true, ".json": true, ".m4a": true,
	".mov": true, ".mp3": true, ".mp4": true, ".odt": true, ".ogg": true, ".otf": true,
	".pdf": true, ".png": true, ".ppt": true, ".pptx": true, ".rar": true, ".rss": true,
	".svg": true, ".tar": true, ".tif": true, ".tiff": true, ".ttf": true, ".txt": true,
	".wav": true, ".webm": true, ".webp": true, ".woff": true, ".woff2": true,
	".xls": true, ".xlsx": true, ".xml": true, ".zip": true,
}

// IsResourceURL reports whether u names a file that is not an HTML page,
// judged by its extension. Such URLs are checked with HEAD.
func IsResourceURL(u *url.URL) bool {
	return resourceExtensions[strings.ToLower(path.Ext(u.Path))]
}
