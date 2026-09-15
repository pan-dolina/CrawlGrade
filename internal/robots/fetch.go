package robots

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
)

// Outcome summarizes how robots.txt was obtained.
type Outcome string

// Outcomes, with the access policy RFC 9309 attaches to each.
const (
	// OutcomeOK: the file was fetched and its rules apply.
	OutcomeOK Outcome = "ok"
	// OutcomeNotFound: 4xx; crawlers may access everything.
	OutcomeNotFound Outcome = "not-found"
	// OutcomeUnavailable: 5xx, network failure or redirect loop; crawlers
	// must assume everything is disallowed.
	OutcomeUnavailable Outcome = "unavailable"
	// OutcomeBlocked: the network policy refused the request (for example a
	// redirect to a private address). Treated like unavailable.
	OutcomeBlocked Outcome = "blocked"
)

// fetchLimit is the largest robots.txt body read. The first MaxParseBytes
// are parsed; files between the two sizes are reported but still used.
const fetchLimit = 2 << 20

// Fetcher performs requests; *fetcher.Client implements it.
type Fetcher interface {
	Fetch(ctx context.Context, req fetcher.Request) (*fetcher.Response, error)
}

// File is the robots.txt of one site origin.
type File struct {
	URL        string   `json:"url"`
	FinalURL   string   `json:"final_url,omitempty"`
	Status     int      `json:"status,omitempty"`
	Outcome    Outcome  `json:"outcome"`
	Error      string   `json:"error,omitempty"`
	Size       int      `json:"size"`
	Sitemaps   []string `json:"sitemaps,omitempty"`
	RuleCount  int      `json:"rules"`
	Truncated  bool     `json:"truncated,omitempty"`
	CrawlDelay float64  `json:"crawl_delay_seconds,omitempty"`
	Robots     *Robots  `json:"-"`
}

// Fetch retrieves robots.txt for the origin of site.
func Fetch(ctx context.Context, f Fetcher, site *url.URL) *File {
	u := &url.URL{Scheme: site.Scheme, Host: site.Host, Path: robotsTxtPath}
	file := &File{URL: u.String()}
	resp, err := f.Fetch(ctx, fetcher.Request{
		URL:                  file.URL,
		Accept:               "text/plain,*/*;q=0.5",
		MaxBodyBytes:         fetchLimit,
		MaxDecompressedBytes: fetchLimit,
	})
	if resp != nil {
		file.FinalURL = resp.FinalURL
		file.Status = resp.Status
	}
	switch {
	case errors.Is(err, netguard.ErrBlocked):
		file.Outcome, file.Error = OutcomeBlocked, err.Error()
	case err != nil:
		file.Outcome, file.Error = OutcomeUnavailable, err.Error()
	case resp.Status >= 200 && resp.Status < 300:
		file.Outcome = OutcomeOK
		file.Size = len(resp.Body)
		file.Robots = Parse(resp.Body)
	case resp.Status >= 400 && resp.Status < 500:
		file.Outcome = OutcomeNotFound
	case resp.Status >= 300 && resp.Status < 400:
		// A redirect without Location; RFC 9309 treats it as unreachable
		// after five hops, and a response we cannot follow as unavailable.
		file.Outcome, file.Error = OutcomeUnavailable, fmt.Sprintf("redirect status %d without a usable Location", resp.Status)
	default:
		file.Outcome = OutcomeUnavailable
		file.Error = fmt.Sprintf("HTTP status %d", resp.Status)
	}
	if file.Robots != nil {
		for _, s := range file.Robots.Sitemaps {
			file.Sitemaps = append(file.Sitemaps, s.URL)
		}
		file.RuleCount = file.Robots.RuleCount
		file.Truncated = file.Robots.Truncated
		if d, ok := file.Robots.CrawlDelay(ProductToken); ok {
			file.CrawlDelay = d.Seconds()
		}
	}
	return file
}

// Allowed reports whether CrawlGrade may fetch u under this file.
func (f *File) Allowed(u *url.URL) bool {
	return f.AllowedFor(ProductToken, u)
}

// AllowedFor reports whether agent may fetch u under this file.
func (f *File) AllowedFor(agent string, u *url.URL) bool {
	switch f.Outcome {
	case OutcomeOK:
		return f.Robots.Allowed(agent, requestURI(u))
	case OutcomeNotFound:
		return true
	}
	return u.EscapedPath() == robotsTxtPath
}

func requestURI(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	return p
}

// Findings reports problems with the file for the site whose start URL is
// start.
func (f *File) Findings(start *url.URL) []findings.Finding {
	var out []findings.Finding
	switch f.Outcome {
	case OutcomeNotFound:
		out = append(out, findings.RobotsNotFound.New(f.URL, fmt.Sprintf("HTTP status %d", f.Status)))
		return out
	case OutcomeUnavailable, OutcomeBlocked:
		ev := []string{fmt.Sprintf("outcome: %s", f.Outcome)}
		if f.Status != 0 {
			ev = append(ev, fmt.Sprintf("HTTP status %d", f.Status))
		}
		if f.Error != "" {
			ev = append(ev, f.Error)
		}
		out = append(out, findings.RobotsUnavailable.New(f.URL, ev...))
		return out
	}
	r := f.Robots
	uri := requestURI(start)
	for _, agent := range []string{"*", "googlebot", "bingbot"} {
		if ok, rule := r.Match(agent, uri); !ok {
			out = append(out, findings.RobotsBlocksStart.New(start.String(),
				fmt.Sprintf("user-agent %s: line %d: Disallow: %s", agent, rule.Line, rule.Pattern)))
			break
		}
	}
	if ok, rule := r.Match(ProductToken, uri); !ok && len(out) == 0 {
		out = append(out, findings.RobotsBlocksCrawler.New(start.String(),
			fmt.Sprintf("line %d: Disallow: %s", rule.Line, rule.Pattern)))
	}
	if len(r.Warnings) > 0 {
		var ev []string
		for _, w := range r.Warnings {
			ev = append(ev, fmt.Sprintf("line %d: %s", w.Line, w.Message))
		}
		out = append(out, findings.RobotsSyntax.New(f.URL, ev...))
	}
	if len(r.Sitemaps) == 0 {
		out = append(out, findings.RobotsNoSitemap.New(f.URL))
	}
	var bad []string
	for _, s := range r.Sitemaps {
		su, err := url.Parse(s.URL)
		if err != nil || !su.IsAbs() || (su.Scheme != "http" && su.Scheme != "https") {
			bad = append(bad, fmt.Sprintf("line %d: %s", s.Line, s.URL))
		}
	}
	if len(bad) > 0 {
		out = append(out, findings.RobotsInvalidSitemap.New(f.URL, bad...))
	}
	if f.Size > MaxParseBytes || r.Truncated {
		out = append(out, findings.RobotsTooLarge.New(f.URL,
			fmt.Sprintf("size %d bytes, %d rules", f.Size, r.RuleCount)))
	}
	if strings.TrimSpace(f.FinalURL) != "" && f.FinalURL != f.URL {
		out = append(out, findings.RobotsRedirected.New(f.URL, "final URL: "+f.FinalURL))
	}
	return out
}
