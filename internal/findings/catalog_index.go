package findings

// Crawlability: robots meta tags and X-Robots-Tag.
var (
	IndexNoindex = register(Rule{
		ID: "SEO-INDEX-001", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Page is excluded from indexing",
		Description:    "A robots meta tag or X-Robots-Tag header contains noindex (or none). The page will not appear in search results.",
		Recommendation: "Confirm the page is meant to be excluded; remove noindex from pages that should rank.",
	})
	IndexNofollow = register(Rule{
		ID: "SEO-INDEX-002", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Page asks crawlers not to follow its links",
		Description:    "A page-level nofollow (or none) directive was found; links on the page pass no signals.",
		Recommendation: "Use page-level nofollow only for pages whose links should not be followed.",
	})
	IndexConflict = register(Rule{
		ID: "SEO-INDEX-003", Category: CategoryCrawlability, Severity: SeverityMedium,
		Title:          "Robots directives conflict",
		Description:    "The page declares both index and noindex, or follow and nofollow. Search engines apply the most restrictive directive.",
		Recommendation: "Remove the contradicting directive so the intent is explicit.",
	})
	IndexInvalidDirective = register(Rule{
		ID: "SEO-INDEX-004", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Robots metadata contains invalid directives",
		Description:    "Unknown directives or invalid values (for example max-snippet:abc or max-image-preview:huge) are ignored by search engines.",
		Recommendation: "Use documented directives: index, noindex, follow, nofollow, none, noarchive, nosnippet, max-snippet:N, max-image-preview:none|standard|large, max-video-preview:N.",
	})
	IndexSnippetRestricted = register(Rule{
		ID: "SEO-INDEX-005", Category: CategoryCrawlability, Severity: SeverityInfo,
		Title:          "Search result presentation is restricted",
		Description:    "nosnippet, max-snippet:0, max-image-preview:none or noarchive limit how the page is shown in search results.",
		Recommendation: "Keep these directives only where the restriction is intended.",
	})
	IndexStartNoindex = register(Rule{
		ID: "SEO-INDEX-006", Category: CategoryCrawlability, Severity: SeverityCritical,
		Title:          "Start page is excluded from indexing",
		Description:    "The audited start URL carries noindex. For a home page this usually removes the site's most important page from search results.",
		Recommendation: "Remove noindex from the start page unless the whole site is intentionally hidden (for example a staging environment).",
	})
	IndexNoindexCanonical = register(Rule{
		ID: "SEO-INDEX-007", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Noindex page declares a different canonical URL",
		Description:    "noindex combined with a canonical pointing elsewhere sends mixed signals; the noindex may be passed to the canonical target.",
		Recommendation: "Use either a canonical to the preferred URL or noindex, not both.",
	})
)
