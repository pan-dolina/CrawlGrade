package findings

var (
	HTTPFailure    = register(Rule{ID: "SEO-HTTP-001", Category: CategoryCrawlability, Severity: SeverityHigh, Title: "URL could not be fetched successfully", Description: "An observed request failed or returned an error status.", Recommendation: "Inspect the response and repair unavailable URLs."})
	HTTPRedirect   = register(Rule{ID: "SEO-HTTP-002", Category: CategoryCrawlability, Severity: SeverityInfo, Title: "URL redirects", Description: "The requested URL redirects before returning content.", Recommendation: "Link directly to the final URL where appropriate."})
	HTMLParse      = register(Rule{ID: "SEO-HTTP-003", Category: CategoryCrawlability, Severity: SeverityMedium, Title: "HTML could not be parsed", Description: "The HTML parser could not produce a page model.", Recommendation: "Simplify malformed or excessively nested markup."})
	SitemapTarget  = register(Rule{ID: "SEO-SITEMAP-010", Category: CategoryCrawlability, Severity: SeverityMedium, Title: "Sitemap URL is not an indexable canonical destination", Description: "An observed sitemap URL fails, redirects, is disallowed, is noindex or declares a different canonical.", Recommendation: "List only canonical, indexable URLs in sitemaps."})
	ExternalBroken = register(Rule{ID: "SEO-LINK-008", Category: CategoryInternalLinking, Severity: SeverityLow, Title: "External link check failed", Description: "A bounded HEAD check failed; some sites reject HEAD requests.", Recommendation: "Verify the destination before changing the link."})
)
