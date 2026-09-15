// Package fetcher implements the bounded HTTP client used for every request
// CrawlGrade makes.
//
// The client never follows redirects on its own: each hop is issued, checked
// and recorded by Fetch, so that loops, chains and unsafe destinations are
// visible to callers. Response bodies are read through size limits both on
// the wire and after decompression, which protects against giant responses
// and compression bombs.
package fetcher

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default limits. See docs/crawling.md.
const (
	DefaultTimeout              = 15 * time.Second
	DefaultMaxBodyBytes         = 5 << 20
	DefaultMaxDecompressedBytes = 10 << 20
	DefaultMaxRedirects         = 10
	maxHeaderBytes              = 64 << 10
	maxDrainBytes               = 64 << 10
)

// Errors returned by Fetch. They are wrapped; use errors.Is.
var (
	ErrRedirectLoop        = errors.New("redirect loop")
	ErrTooManyRedirects    = errors.New("too many redirects")
	ErrUnsupportedURL      = errors.New("unsupported URL")
	ErrUnsupportedEncoding = errors.New("unsupported content encoding")
	ErrTooLarge            = errors.New("response exceeds size limit")
)

// Options configures a Client.
type Options struct {
	UserAgent            string
	Timeout              time.Duration // whole request including redirects and body
	MaxBodyBytes         int64         // bytes read from the connection
	MaxDecompressedBytes int64         // bytes after content decoding
	MaxRedirects         int
	// DialContext establishes connections. It is where the network policy
	// is enforced; nil uses a plain net.Dialer (tests only).
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	// Wait is called before every request, including redirect hops, and is
	// used for rate limiting. It may be nil.
	Wait func(ctx context.Context) error
	// MaxConnsPerHost bounds parallel connections to one host.
	MaxConnsPerHost int
	// TLSConfig overrides the TLS configuration (tests only).
	TLSConfig *tls.Config
}

func (o *Options) setDefaults() {
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if o.MaxDecompressedBytes <= 0 {
		o.MaxDecompressedBytes = DefaultMaxDecompressedBytes
	}
	if o.MaxRedirects < 0 {
		o.MaxRedirects = 0
	} else if o.MaxRedirects == 0 {
		o.MaxRedirects = DefaultMaxRedirects
	}
	if o.MaxConnsPerHost <= 0 {
		o.MaxConnsPerHost = 4
	}
	if o.UserAgent == "" {
		o.UserAgent = "CrawlGrade"
	}
}

// Client performs bounded requests. It is safe for concurrent use.
type Client struct {
	opts Options
	http *http.Client
}

// New returns a Client.
func New(opts Options) *Client {
	opts.setDefaults()
	dial := opts.DialContext
	if dial == nil {
		d := &net.Dialer{Timeout: 10 * time.Second}
		dial = d.DialContext
	}
	tlsConfig := opts.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	transport := &http.Transport{
		// Proxies would connect on our behalf and bypass the dialer's
		// destination checks, so environment proxy settings are ignored.
		Proxy:                  nil,
		DialContext:            dial,
		ForceAttemptHTTP2:      true,
		TLSClientConfig:        tlsConfig,
		MaxIdleConns:           64,
		MaxIdleConnsPerHost:    opts.MaxConnsPerHost,
		MaxConnsPerHost:        opts.MaxConnsPerHost,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  opts.Timeout,
		ExpectContinueTimeout:  time.Second,
		MaxResponseHeaderBytes: maxHeaderBytes,
		// Decompression is done by Fetch under explicit limits.
		DisableCompression: true,
	}
	return &Client{
		opts: opts,
		http: &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Close releases idle connections.
func (c *Client) Close() {
	c.http.CloseIdleConnections()
}

// Request describes one fetch.
type Request struct {
	URL    string
	Method string // GET (default) or HEAD
	Accept string
	// Per-request overrides of the client's body limits (0 = client default).
	MaxBodyBytes         int64
	MaxDecompressedBytes int64
}

// Hop is one redirect response.
type Hop struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Location string `json:"location"`
}

// Response is the outcome of a fetch. On error, fields describing the part
// of the exchange that succeeded (status, headers, redirects) are still set.
type Response struct {
	RequestedURL    string
	FinalURL        string
	Status          int
	Header          http.Header
	Redirects       []Hop
	ContentType     string // media type, lower case, without parameters
	ContentEncoding string
	Body            []byte // decoded body
	WireBytes       int64  // body bytes read from the connection
	Duration        time.Duration
}

// LimitError reports which limit a response exceeded.
type LimitError struct {
	Limit string // "body" or "decompressed"
	Max   int64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("response exceeds %s size limit of %d bytes", e.Limit, e.Max)
}

// Unwrap makes LimitError match ErrTooLarge.
func (e *LimitError) Unwrap() error { return ErrTooLarge }

// Fetch performs the request, following redirects manually.
func (c *Client) Fetch(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodHead {
		return nil, fmt.Errorf("%w: method %s", ErrUnsupportedURL, method)
	}
	resp := &Response{RequestedURL: req.URL}
	defer func() { resp.Duration = time.Since(start) }()

	current, err := ParseHTTPURL(req.URL)
	if err != nil {
		return resp, err
	}
	seen := map[string]bool{}
	for {
		resp.FinalURL = current.String()
		if seen[resp.FinalURL] {
			return resp, fmt.Errorf("%w at %s", ErrRedirectLoop, resp.FinalURL)
		}
		seen[resp.FinalURL] = true

		if c.opts.Wait != nil {
			if err := c.opts.Wait(ctx); err != nil {
				return resp, err
			}
		}
		hr, err := c.do(ctx, method, current, req.Accept)
		if err != nil {
			return resp, err
		}
		resp.Status = hr.StatusCode
		resp.Header = hr.Header

		if isRedirect(hr.StatusCode) {
			loc := hr.Header.Get("Location")
			drainAndClose(hr.Body)
			if loc == "" {
				// A redirect status without Location is the final response.
				return resp, nil
			}
			resp.Redirects = append(resp.Redirects, Hop{URL: resp.FinalURL, Status: hr.StatusCode, Location: loc})
			next, err := resolveLocation(current, loc)
			if err != nil {
				return resp, err
			}
			if len(resp.Redirects) > c.opts.MaxRedirects {
				resp.FinalURL = next.String()
				return resp, fmt.Errorf("%w (limit %d)", ErrTooManyRedirects, c.opts.MaxRedirects)
			}
			if hr.StatusCode == http.StatusSeeOther && method != http.MethodHead {
				method = http.MethodGet
			}
			current = next
			continue
		}
		err = c.readBody(hr, req, resp)
		return resp, err
	}
}

func (c *Client) do(ctx context.Context, method string, u *url.URL, accept string) (*http.Response, error) {
	hreq, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedURL, err)
	}
	hreq.Header.Set("User-Agent", c.opts.UserAgent)
	hreq.Header.Set("Accept-Encoding", "gzip")
	if accept == "" {
		accept = "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5"
	}
	hreq.Header.Set("Accept", accept)
	return c.http.Do(hreq)
}

func (c *Client) readBody(hr *http.Response, req Request, resp *Response) error {
	defer hr.Body.Close()
	resp.ContentEncoding = strings.ToLower(strings.TrimSpace(hr.Header.Get("Content-Encoding")))
	if mt, _, err := mime.ParseMediaType(hr.Header.Get("Content-Type")); err == nil {
		resp.ContentType = strings.ToLower(mt)
	}
	if hr.Request != nil && hr.Request.Method == http.MethodHead {
		return nil
	}
	maxBody := c.opts.MaxBodyBytes
	if req.MaxBodyBytes > 0 {
		maxBody = req.MaxBodyBytes
	}
	maxDecoded := c.opts.MaxDecompressedBytes
	if req.MaxDecompressedBytes > 0 {
		maxDecoded = req.MaxDecompressedBytes
	}
	if hr.ContentLength > maxBody {
		return &LimitError{Limit: "body", Max: maxBody}
	}

	wire := &countingReader{r: io.LimitReader(hr.Body, maxBody+1)}
	var body io.Reader = wire
	switch resp.ContentEncoding {
	case "", "identity":
	case "gzip", "x-gzip":
		gz, err := gzip.NewReader(wire)
		if err != nil {
			if wire.n > maxBody {
				return &LimitError{Limit: "body", Max: maxBody}
			}
			return fmt.Errorf("invalid gzip body: %w", err)
		}
		defer gz.Close()
		body = gz
	default:
		return fmt.Errorf("%w %q", ErrUnsupportedEncoding, resp.ContentEncoding)
	}
	data, err := io.ReadAll(io.LimitReader(body, maxDecoded+1))
	resp.WireBytes = wire.n
	switch {
	case wire.n > maxBody:
		return &LimitError{Limit: "body", Max: maxBody}
	case int64(len(data)) > maxDecoded:
		return &LimitError{Limit: "decompressed", Max: maxDecoded}
	case err != nil:
		return fmt.Errorf("reading body: %w", err)
	}
	resp.Body = data
	return nil
}

// ParseHTTPURL parses an absolute http or https URL. URLs with credentials
// are rejected: a site must not be able to make CrawlGrade send them.
func ParseHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedURL, err)
	}
	if err := checkURL(u); err != nil {
		return nil, err
	}
	return u, nil
}

func checkURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q", ErrUnsupportedURL, u.Scheme)
	}
	if u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("%w: missing host", ErrUnsupportedURL)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials in URL", ErrUnsupportedURL)
	}
	return nil
}

func resolveLocation(base *url.URL, loc string) (*url.URL, error) {
	ref, err := url.Parse(strings.TrimSpace(loc))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid redirect location %q", ErrUnsupportedURL, loc)
	}
	next := base.ResolveReference(ref)
	next.Fragment = ""
	next.RawFragment = ""
	if err := checkURL(next); err != nil {
		return nil, fmt.Errorf("redirect to %s: %w", next.Redacted(), err)
	}
	return next, nil
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxDrainBytes))
	_ = body.Close()
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
