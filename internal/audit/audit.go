// Package audit wires the crawl and the analyses together into a single
// report. It is the only place that knows how to turn a URL into a report: it
// fetches robots.txt and a sitemap, crawls the site while extracting each
// page, runs the per-page and site-wide analyses, and assembles their findings
// into a report.
//
// The audit is a pure function of its inputs (the start URL, the fetcher and
// the options) plus the network. It is deterministic for a stable site: the
// crawl is level-synchronous and every analysis sorts its findings.
package audit

import (
	"context"
	"github.com/pan-dolina/crawlgrade/internal/duplicates"
	"github.com/pan-dolina/crawlgrade/internal/structureddata"
	"net/url"
	"sort"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/content"
	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/htmlcheck"
	"github.com/pan-dolina/crawlgrade/internal/links"
	"github.com/pan-dolina/crawlgrade/internal/report"
	"github.com/pan-dolina/crawlgrade/internal/robots"
	"github.com/pan-dolina/crawlgrade/internal/sitemap"
	"github.com/pan-dolina/crawlgrade/internal/terms_strength"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
	"github.com/pan-dolina/crawlgrade/internal/webhygiene"
)

// Options configures an audit.
type Options struct {
	// MaxPages and Concurrency default when zero. MaxDepth zero audits only
	// the start URL; callers wanting deeper crawls must set it explicitly.
	MaxDuration time.Duration
	MaxPages    int
	MaxDepth    int
	Concurrency int
	// AllowPrivate permits auditing local development sites.
	AllowPrivate       bool
	Keywords           []string
	CheckExternalLinks bool
}

// Result is the outcome of an audit.
type Result struct {
	// Report is the assembled report. It is always non-nil.
	Report *report.Report
	Crawl  *crawler.Result
	Err    error
}

// Run audits start and returns its report. The fetcher performs every request;
// it must embed the network policy.
func Run(ctx context.Context, f crawler.Fetcher, start *url.URL, opts Options) *Result {
	duration := opts.MaxDuration
	if duration <= 0 {
		duration = crawler.DefaultMaxDuration
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	// robots.txt and the sitemap are fetched before the crawl so their rules
	// govern the crawl and their findings feed the report.
	robotsFile := robots.Fetch(ctx, f, start)
	seeds := []sitemap.Seed{sitemap.DefaultSeed(start)}
	for _, loc := range robotsFile.Sitemaps {
		seeds = append(seeds, sitemap.Seed{URL: loc, Source: "robots.txt", Declared: true})
	}
	sitemapResult := sitemap.Collect(ctx, f, seeds, urlnorm.NewScope(start), sitemap.DefaultLimits())

	var seedsToCrawl []*url.URL
	for _, entry := range sitemapResult.URLs {
		if u, err := urlnorm.Parse(entry.Loc); err == nil {
			seedsToCrawl = append(seedsToCrawl, u)
		}
	}
	origin := func(u *url.URL) string { return u.Scheme + "://" + u.Host }
	policies := map[string]*robots.File{origin(start): robotsFile}
	allowed := func(u *url.URL) bool {
		key := origin(u)
		file := policies[key]
		if file == nil {
			file = robots.Fetch(ctx, f, u)
			policies[key] = file
		}
		return file.Allowed(u)
	}
	crawl := crawler.Crawl(ctx, f, start, crawler.Options{
		MaxPages:    defaultInt(opts.MaxPages, crawler.DefaultMaxPages),
		MaxDuration: duration,
		MaxDepth:    opts.MaxDepth,
		Concurrency: defaultInt(opts.Concurrency, crawler.DefaultConcurrency),
		TrapLimits:  urlnorm.DefaultTrapLimits(),
		Allowed:     allowed,
		Process:     processPage,
		Seeds:       seedsToCrawl,
	})

	groups := map[string][]findings.Finding{
		report.GroupCrawl:      append(robotsFile.Findings(start), sitemapResult.Findings()...),
		report.GroupMetadata:   metadataFindings(crawl),
		report.GroupContent:    contentFindings(crawl),
		report.GroupStructured: structuredFindings(crawl),
		report.GroupLinking:    linkingFindings(crawl),
		report.GroupWebHygiene: webHygieneFindings(crawl),
	}
	var origins []string
	for key := range policies {
		if key != origin(start) {
			origins = append(origins, key)
		}
	}
	sort.Strings(origins)
	for _, key := range origins {
		u, _ := url.Parse(key)
		groups[report.GroupCrawl] = append(groups[report.GroupCrawl], policies[key].Findings(u)...)
	}
	groups[report.GroupContent] = append(groups[report.GroupContent], termFindings(crawl)...)

	var dupPages []*duplicates.Page
	for _, c := range contentResults(crawl) {
		if !c.NoText {
			dupPages = append(dupPages, duplicates.NewPage(c))
		}
	}
	groups[report.GroupContent] = append(groups[report.GroupContent], duplicates.Analyze(dupPages).Findings()...)
	groups[report.GroupCrawl] = append(groups[report.GroupCrawl], crawlFindings(crawl, sitemapResult, robotsFile)...)
	groups[report.GroupLinking] = append(groups[report.GroupLinking], observedLinkFindings(crawl)...)
	if opts.CheckExternalLinks {
		groups[report.GroupLinking] = append(groups[report.GroupLinking], checkExternal(ctx, f, crawl)...)
	}
	if start.Scheme == "https" && len(crawl.Pages) > 0 && ctx.Err() == nil {
		httpURL := *start
		httpURL.Scheme = "http"
		resp, err := f.Fetch(ctx, fetcher.Request{URL: httpURL.String(), Method: "HEAD"})
		if err != nil || resp == nil {
			groups[report.GroupWebHygiene] = append(groups[report.GroupWebHygiene], findings.WebHygieneRedirect.New(start.String(), "HTTP redirect check unavailable"))
		} else if len(resp.Redirects) == 0 || !isHTTPS(resp.FinalURL) {
			groups[report.GroupWebHygiene] = append(groups[report.GroupWebHygiene], findings.WebHygieneRedirect.New(start.String(), resp.FinalURL))
		}
	}
	rep := report.New(start.String(), crawl, groups)
	enrichReport(rep, crawl, opts.Keywords)
	return &Result{Report: rep, Crawl: crawl, Err: ctx.Err()}
}

// defaultInt returns v when it is positive, otherwise def. The audit options
// use the zero value for "unset", which the crawler would otherwise treat as
// "fetch only the start URL".
func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// parsedPage is the analysis result stored on crawler.Page.Data. It carries
// the parse scope, which CanonicalFindings needs and which is not otherwise
// reachable from the exported Page model.
type parsedPage struct {
	Page       *htmlcheck.Page
	Scope      urlnorm.Scope
	Content    *content.Result
	Structured []findings.Finding
	Hygiene    []findings.Finding
}

// processPage is the crawler Process callback. It parses each HTML page with
// htmlcheck, stores the result on Page.Data for the site-wide analyses, and
// returns the in-scope links to follow.
func processPage(_ context.Context, p *crawler.Page) []*url.URL {
	if p.Response == nil || p.Response.Body == nil {
		return nil
	}
	final := p.Response.FinalURL
	if final == "" {
		final = p.URL
	}
	base, err := url.Parse(final)
	if err != nil {
		return nil
	}
	requested, _ := url.Parse(p.URL)
	scope := urlnorm.NewScope(requested)
	hp, root, err := htmlcheck.Parse(p.Response.Body, base, p.Response.Header.Get("Content-Type"), scope)
	if err != nil {
		return nil
	}
	hp.AddRobotsHeaders(p.Response.Header.Values("X-Robots-Tag"))
	hp.AddHeaderCanonicals(p.Response.Header.Values("Link"))
	sd := structureddata.Extract(root, base)
	sf := append(sd.Findings(hp.URL), sd.BreadcrumbFindings(hp.URL)...)
	p.Data = parsedPage{Page: hp, Scope: scope, Content: content.ExtractTree(hp.URL, root), Structured: sf, Hygiene: webhygiene.MarkupFindings(hp.URL, root)}
	return followLinks(hp)
}

// followLinks returns the in-scope links of a parsed page. htmlcheck already
// resolved each href and marked external links, so internal links are those
// whose External flag is false.
func followLinks(hp *htmlcheck.Page) []*url.URL {
	var out []*url.URL
	for _, l := range hp.Links {
		if l.External || l.URL == "" {
			continue
		}
		u, err := url.Parse(l.URL)
		if err != nil {
			continue
		}
		out = append(out, u)
	}
	for _, img := range hp.Images {
		if !img.External && img.Src != "" {
			if u, err := urlnorm.Parse(img.Src); err == nil {
				out = append(out, u)
			}
		}
	}
	return out
}

// metadataFindings runs the per-page metadata checks and the site-wide
// duplicate-metadata checks.
func metadataFindings(crawl *crawler.Result) []findings.Finding {
	var out []findings.Finding
	pages := parsedPages(crawl)
	start := ""
	if len(crawl.Pages) > 0 {
		start = crawl.Pages[0].URL
		if crawl.Pages[0].Response != nil && crawl.Pages[0].Response.FinalURL != "" {
			start = crawl.Pages[0].Response.FinalURL
		}
	}
	for _, pp := range pages {
		hp := pp.Page
		out = append(out, hp.MetadataFindings()...)
		out = append(out, hp.HeadingFindings(htmlcheck.CheckOptions{})...)
		out = append(out, hp.CanonicalFindings(pp.Scope)...)
		out = append(out, hp.RobotsFindings(hp.URL == start)...)
		out = append(out, hp.HreflangFindings()...)
		out = append(out, hp.LangFindings()...)
		out = append(out, hp.SocialFindings()...)
		out = append(out, hp.SocialRequiredFindings()...)
		out = append(out, hp.ImageFindings()...)
		out = append(out, hp.AnchorFindings()...)
	}
	if len(pages) > 0 {
		hpPages := make([]*htmlcheck.Page, len(pages))
		for i, pp := range pages {
			hpPages[i] = pp.Page
		}
		out = append(out, htmlcheck.DuplicateMetadataFindings(hpPages)...)
		out = append(out, htmlcheck.DuplicateH1Findings(hpPages)...)
		out = append(out, htmlcheck.HreflangSiteFindings(hpPages)...)
	}
	return out
}

// contentFindings runs the per-page content extraction findings.
func contentFindings(crawl *crawler.Result) []findings.Finding {
	var out []findings.Finding
	for _, c := range contentResults(crawl) {
		out = append(out, c.Findings()...)
	}
	return out
}

// termFindings runs the site-wide term-strength analysis.
func termFindings(crawl *crawler.Result) []findings.Finding {
	pages := contentResults(crawl)
	if len(pages) == 0 {
		return nil
	}
	return terms_strength.Analyze(pages).Findings()
}

// structuredFindings collects checks performed before response bodies were dropped.
func structuredFindings(crawl *crawler.Result) []findings.Finding {
	var out []findings.Finding
	for _, p := range parsedPages(crawl) {
		out = append(out, p.Structured...)
	}
	return out
}

// linkingFindings runs the site-wide link-graph analysis.
func linkingFindings(crawl *crawler.Result) []findings.Finding {
	pages := linkPages(crawl)
	if len(pages) == 0 {
		return nil
	}
	return links.Analyze(pages).Findings()
}

// webHygieneFindings runs the passive web hygiene checks.
func webHygieneFindings(crawl *crawler.Result) []findings.Finding {
	checks := hygieneChecks(crawl)
	if len(checks) == 0 {
		return nil
	}
	out := webhygiene.Findings(checks)
	for _, p := range parsedPages(crawl) {
		out = append(out, p.Hygiene...)
	}
	return out
}

// contentResults returns the extracted content of every crawled page.
func contentResults(crawl *crawler.Result) []*content.Result {
	var out []*content.Result
	for _, p := range parsedPages(crawl) {
		if p.Content != nil && p.Page.Robots.Indexable() {
			out = append(out, p.Content)
		}
	}
	return out
}

// parsedPages returns the parsed pages of the crawl, in order.
func parsedPages(crawl *crawler.Result) []parsedPage {
	var out []parsedPage
	for _, p := range crawl.Pages {
		pp, ok := p.Data.(parsedPage)
		if !ok || pp.Page == nil {
			continue
		}
		out = append(out, pp)
	}
	return out
}

// linkPages builds the links.Page inputs from the crawl.
func linkPages(crawl *crawler.Result) []*links.Page {
	var out []*links.Page
	for i, p := range crawl.Pages {
		lp := &links.Page{URL: p.URL, Failed: p.Err != nil, Start: i == 0}
		if p.Response != nil {
			lp.Status = p.Response.Status
		}
		if pp, ok := p.Data.(parsedPage); ok {
			lp.URL = pp.Page.URL
			lp.Links = toLinkValues(pp.Page.Links)
			lp.Indexable = pp.Page.Robots.Indexable()
		}
		out = append(out, lp)
	}
	return out
}

// hygieneChecks builds the web hygiene inputs from the crawl.
func hygieneChecks(crawl *crawler.Result) []webhygiene.Check {
	var checks []webhygiene.Check
	for _, p := range crawl.Pages {
		if p.Response == nil {
			continue
		}
		checks = append(checks, webhygiene.Check{
			URL:     p.URL,
			Header:  p.Response.Header,
			Body:    string(p.Response.Body),
			IsHTTPS: isHTTPS(p.Response.FinalURL),
			Status:  p.Response.Status,
		})
	}
	return checks
}

// toLinkValues converts htmlcheck links to links.Page link values.
func toLinkValues(hpLinks []htmlcheck.Link) []links.Link {
	var out []links.Link
	for _, l := range hpLinks {
		out = append(out, links.Link{
			URL:        l.URL,
			External:   l.External,
			Resource:   l.Resource,
			AnchorText: l.AnchorText,
			Empty:      l.Empty,
			NoFollow:   l.NoFollow,
		})
	}
	return out
}

// isHTTPS reports whether u uses the https scheme.
func isHTTPS(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https"
}
