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
	"net/url"

	"github.com/pan-dolina/crawlgrade/internal/content"
	"github.com/pan-dolina/crawlgrade/internal/crawler"
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
	// MaxPages, MaxDepth and Concurrency mirror the crawler limits. The zero
	// values are the crawler defaults.
	MaxPages    int
	MaxDepth    int
	Concurrency int
	// AllowPrivate permits auditing local development sites.
	AllowPrivate bool
}

// Result is the outcome of an audit.
type Result struct {
	// Report is the assembled report. It is always non-nil.
	Report *report.Report
}

// Run audits start and returns its report. The fetcher performs every request;
// it must embed the network policy.
func Run(ctx context.Context, f crawler.Fetcher, start *url.URL, opts Options) *Result {
	// robots.txt and the sitemap are fetched before the crawl so their rules
	// govern the crawl and their findings feed the report.
	robotsFile := robots.Fetch(ctx, f, start)
	sitemapResult := sitemap.Collect(ctx, f, []sitemap.Seed{sitemap.DefaultSeed(start)}, urlnorm.NewScope(start), sitemap.DefaultLimits())

	allowed := allowedFunc(robotsFile)
	crawl := crawler.Crawl(ctx, f, start, crawler.Options{
		MaxPages:    defaultInt(opts.MaxPages, crawler.DefaultMaxPages),
		MaxDepth:    defaultInt(opts.MaxDepth, crawler.DefaultMaxDepth),
		Concurrency: defaultInt(opts.Concurrency, crawler.DefaultConcurrency),
		TrapLimits:  urlnorm.DefaultTrapLimits(),
		Allowed:     allowed,
		Process:     processPage,
	})

	groups := map[string][]findings.Finding{
		report.GroupCrawl:      append(robotsFile.Findings(start), sitemapResult.Findings()...),
		report.GroupMetadata:   metadataFindings(crawl),
		report.GroupContent:    contentFindings(crawl),
		report.GroupStructured: structuredFindings(crawl),
		report.GroupLinking:    linkingFindings(crawl),
		report.GroupWebHygiene: webHygieneFindings(crawl),
	}
	groups[report.GroupContent] = append(groups[report.GroupContent], termFindings(crawl)...)

	return &Result{Report: report.New(start.String(), crawl, groups)}
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

// allowedFunc returns the crawler Allowed callback from the robots.txt outcome.
// A nil callback lets the crawler fetch everything, which matches the robots
// package when the file is missing or unusable.
func allowedFunc(file *robots.File) func(*url.URL) bool {
	if file.Outcome != robots.OutcomeOK {
		return nil
	}
	return file.Allowed
}

// parsedPage is the analysis result stored on crawler.Page.Data. It carries
// the parse scope, which CanonicalFindings needs and which is not otherwise
// reachable from the exported Page model.
type parsedPage struct {
	Page  *htmlcheck.Page
	Scope urlnorm.Scope
}

// processPage is the crawler Process callback. It parses each HTML page with
// htmlcheck, stores the result on Page.Data for the site-wide analyses, and
// returns the in-scope links to follow.
func processPage(_ context.Context, p *crawler.Page) []*url.URL {
	if p.Response == nil || p.Response.Body == nil {
		return nil
	}
	base, err := url.Parse(p.URL)
	if err != nil {
		return nil
	}
	scope := urlnorm.NewScope(base)
	hp, _, err := htmlcheck.Parse(p.Response.Body, base, p.Response.ContentType, scope)
	if err != nil {
		return nil
	}
	p.Data = parsedPage{Page: hp, Scope: scope}
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
	}
	for _, pp := range pages {
		hp := pp.Page
		out = append(out, hp.MetadataFindings()...)
		out = append(out, hp.HeadingFindings(htmlcheck.CheckOptions{})...)
		out = append(out, hp.CanonicalFindings(pp.Scope)...)
		out = append(out, hp.RobotsFindings(hp.URL == start)...)
		out = append(out, hp.HreflangFindings()...)
		out = append(out, hp.SocialFindings()...)
		out = append(out, hp.AnchorFindings()...)
	}
	if len(pages) > 0 {
		hpPages := make([]*htmlcheck.Page, len(pages))
		for i, pp := range pages {
			hpPages[i] = pp.Page
		}
		out = append(out, htmlcheck.DuplicateMetadataFindings(hpPages)...)
		out = append(out, htmlcheck.DuplicateH1Findings(hpPages)...)
	}
	return out
}

// contentFindings runs the per-page content extraction findings.
func contentFindings(crawl *crawler.Result) []findings.Finding {
	var out []findings.Finding
	for _, p := range crawl.Pages {
		if p.Response == nil || p.Response.Body == nil {
			continue
		}
		ext := content.Extract(p.URL, string(p.Response.Body))
		out = append(out, ext.Findings()...)
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

// structuredFindings runs the structured-data checks. It is reserved for the
// next milestone; the group is defined so the report shape is stable.
func structuredFindings(crawl *crawler.Result) []findings.Finding {
	return nil
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
	return webhygiene.Findings(checks)
}

// contentResults returns the extracted content of every crawled page.
func contentResults(crawl *crawler.Result) []*content.Result {
	var out []*content.Result
	for _, p := range crawl.Pages {
		if p.Response == nil || p.Response.Body == nil {
			continue
		}
		out = append(out, content.Extract(p.URL, string(p.Response.Body)))
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
	for _, pp := range parsedPages(crawl) {
		out = append(out, &links.Page{
			URL:       pp.Page.URL,
			Links:     toLinkValues(pp.Page.Links),
			Indexable: pp.Page.Robots.Indexable(),
		})
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
			IsHTTPS: isHTTPS(p.URL),
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
