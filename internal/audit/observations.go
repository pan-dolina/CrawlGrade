package audit

import (
	"context"
	"fmt"
	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/htmlcheck"
	"github.com/pan-dolina/crawlgrade/internal/robots"
	"github.com/pan-dolina/crawlgrade/internal/sitemap"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
	"slices"
)

func crawlFindings(c *crawler.Result, sm *sitemap.Result, rf *robots.File) []findings.Finding {
	var out []findings.Finding
	lookup := map[string]htmlcheck.Target{}
	for _, p := range c.Pages {
		t := htmlcheck.Target{}
		bad := false
		if p.Err != nil {
			t.Error = p.Err.Error()
			bad = true
		}
		if p.Response != nil {
			t.Status = p.Response.Status
			t.FinalURL = p.Response.FinalURL
			t.Redirects = len(p.Response.Redirects)
		}
		if t.Status >= 400 {
			bad = true
		}
		if bad {
			out = append(out, findings.HTTPFailure.New(p.URL, fmt.Sprintf("status: %d; error: %s", t.Status, t.Error)))
		}
		if t.Redirects > 0 {
			out = append(out, findings.HTTPRedirect.New(p.URL, t.FinalURL))
			bad = true
		}
		if pp, ok := p.Data.(parsedPage); ok {
			t.Canonical = pp.Page.CanonicalURL()
			bad = bad || !pp.Page.Robots.Indexable() || t.Canonical != "" && t.Canonical != pp.Page.URL
		} else if crawler.IsHTML(p.Response) && p.Err == nil {
			out = append(out, findings.HTMLParse.New(p.URL))
		}
		lookup[p.URL] = t
		if bad && sm.Contains(p.URL) {
			out = append(out, findings.SitemapTarget.New(p.URL))
		}
	}
	for _, e := range sm.URLs {
		if u, err := urlnorm.Parse(e.Loc); err == nil && !rf.Allowed(u) {
			out = append(out, findings.SitemapTarget.New(e.Loc, "disallowed by robots.txt"))
		}
	}
	var pages []*htmlcheck.Page
	for _, p := range parsedPages(c) {
		pages = append(pages, p.Page)
	}
	out = append(out, htmlcheck.CanonicalSiteFindings(pages, func(u string) (htmlcheck.Target, bool) { t, ok := lookup[u]; return t, ok })...)
	return out
}

func observedLinkFindings(c *crawler.Result) []findings.Finding {
	failed := map[string]bool{}
	for _, p := range c.Pages {
		failed[p.URL] = p.Err != nil || p.Response != nil && p.Response.Status >= 400
	}
	var out []findings.Finding
	for _, p := range parsedPages(c) {
		for _, h := range p.Page.Hreflangs {
			if failed[h.Href] {
				out = append(out, findings.HreflangTargetProblem.New(p.Page.URL, h.Href))
			}
		}
		for _, img := range p.Page.Images {
			if failed[img.Src] {
				out = append(out, findings.LinkBroken.New(p.Page.URL, "image: "+img.Src))
			}
		}
	}
	return out
}

func checkExternal(ctx context.Context, f crawler.Fetcher, c *crawler.Result) []findings.Finding {
	set := map[string]bool{}
	for _, p := range parsedPages(c) {
		for _, l := range p.Page.Links {
			if l.External && l.URL != "" && len(set) < 100 {
				set[l.URL] = true
			}
		}
	}
	var urls []string
	for u := range set {
		urls = append(urls, u)
	}
	slices.Sort(urls)
	var out []findings.Finding
	for _, u := range urls {
		if ctx.Err() != nil {
			break
		}
		r, err := f.Fetch(ctx, fetcher.Request{URL: u, Method: "HEAD"})
		if err != nil || r == nil || r.Status >= 400 {
			out = append(out, findings.ExternalBroken.New(u))
		}
	}
	return out
}
