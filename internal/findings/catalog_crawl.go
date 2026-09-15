package findings

// Crawlability: robots.txt.
var (
	RobotsNotFound = register(Rule{
		ID: "SEO-ROBOTS-001", Category: CategoryCrawlability, Severity: SeverityInfo,
		Title:          "robots.txt not found",
		Description:    "The server answered robots.txt with a 4xx status. Crawlers treat this as permission to crawl everything, so it is not an error.",
		Recommendation: "Optional: publish a robots.txt to declare sitemap locations and keep crawlers out of areas with no search value.",
	})
	RobotsUnavailable = register(Rule{
		ID: "SEO-ROBOTS-002", Category: CategoryCrawlability, Severity: SeverityHigh,
		Title:          "robots.txt unavailable",
		Description:    "robots.txt returned a 5xx status, could not be fetched, or redirected to a blocked destination. RFC 9309 requires crawlers to assume the whole site is disallowed.",
		Recommendation: "Make robots.txt return 200 (or 404 if you have none). Server errors on robots.txt can stop search engines from crawling the site.",
	})
	RobotsBlocksStart = register(Rule{
		ID: "SEO-ROBOTS-003", Category: CategoryCrawlability, Severity: SeverityCritical,
		Title:          "robots.txt disallows the start URL for search engine crawlers",
		Description:    "A Disallow rule matches the audited start URL for all crawlers or for Googlebot or Bingbot.",
		Recommendation: "Remove or narrow the Disallow rule if the page should be crawled. Note that Disallow does not remove already indexed URLs; use noindex for that.",
	})
	RobotsBlocksCrawler = register(Rule{
		ID: "SEO-ROBOTS-004", Category: CategoryCrawlability, Severity: SeverityHigh,
		Title:          "robots.txt disallows CrawlGrade",
		Description:    "robots.txt disallows the start URL for the crawlgrade user-agent token while search engines are allowed. CrawlGrade respects this and the audit is incomplete.",
		Recommendation: "Allow the crawlgrade token for the audit, or audit from an environment where it is permitted.",
	})
	RobotsSyntax = register(Rule{
		ID: "SEO-ROBOTS-005", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "robots.txt contains lines crawlers ignore",
		Description:    "Some lines have no separator, use unknown directives, or appear before any User-agent line. Crawlers skip them, which may not be what was intended.",
		Recommendation: "Fix or remove the listed lines. Rules must follow a User-agent line; directives such as Noindex or Host are not part of RFC 9309.",
	})
	RobotsNoSitemap = register(Rule{
		ID: "SEO-ROBOTS-006", Category: CategoryCrawlability, Severity: SeverityInfo,
		Title:          "robots.txt declares no sitemap",
		Description:    "robots.txt has no Sitemap directive.",
		Recommendation: "Add a Sitemap: line with the absolute URL of your XML sitemap or sitemap index.",
	})
	RobotsInvalidSitemap = register(Rule{
		ID: "SEO-ROBOTS-007", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "robots.txt Sitemap directive is not an absolute URL",
		Description:    "Sitemap directives must contain fully qualified http or https URLs.",
		Recommendation: "Use absolute sitemap URLs, for example Sitemap: https://example.com/sitemap.xml.",
	})
	RobotsTooLarge = register(Rule{
		ID: "SEO-ROBOTS-008", Category: CategoryCrawlability, Severity: SeverityMedium,
		Title:          "robots.txt exceeds parser limits",
		Description:    "The file is larger than 500 KiB or has more rules than CrawlGrade evaluates. Crawlers may ignore rules beyond their limit.",
		Recommendation: "Simplify robots.txt; use wildcards instead of listing individual URLs.",
	})
	RobotsRedirected = register(Rule{
		ID: "SEO-ROBOTS-009", Category: CategoryCrawlability, Severity: SeverityInfo,
		Title:          "robots.txt redirects",
		Description:    "robots.txt is served through a redirect. Crawlers follow a limited number of redirects and apply the target's rules.",
		Recommendation: "Serve robots.txt directly on each host, or make sure the redirect target is the intended file.",
	})
)

// Crawlability: XML sitemaps.
var (
	SitemapMissing = register(Rule{
		ID: "SEO-SITEMAP-001", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "No usable XML sitemap found",
		Description:    "robots.txt declares no sitemap and /sitemap.xml does not return a valid sitemap.",
		Recommendation: "Publish an XML sitemap listing canonical, indexable URLs and reference it from robots.txt.",
	})
	SitemapUnavailable = register(Rule{
		ID: "SEO-SITEMAP-002", Category: CategoryCrawlability, Severity: SeverityMedium,
		Title:          "Declared sitemap could not be fetched",
		Description:    "A sitemap listed in robots.txt or a sitemap index returned an error status, failed to download or exceeded a safety limit.",
		Recommendation: "Make sure every declared sitemap URL returns 200 with the sitemap document, or remove stale references.",
	})
	SitemapInvalid = register(Rule{
		ID: "SEO-SITEMAP-003", Category: CategoryCrawlability, Severity: SeverityMedium,
		Title:          "Sitemap is not a valid sitemap document",
		Description:    "The file is not well-formed UTF-8 XML or its root element is neither urlset nor sitemapindex.",
		Recommendation: "Serve a sitemaps.org 0.9 document; validate it against the protocol schema.",
	})
	SitemapTooLarge = register(Rule{
		ID: "SEO-SITEMAP-004", Category: CategoryCrawlability, Severity: SeverityMedium,
		Title:          "Sitemap exceeds protocol limits",
		Description:    "The file lists more than 50 000 entries. Search engines ignore entries beyond the limit.",
		Recommendation: "Split the sitemap into files of at most 50 000 URLs and 50 MB (uncompressed) referenced by a sitemap index.",
	})
	SitemapWarnings = register(Rule{
		ID: "SEO-SITEMAP-005", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Sitemap contains malformed entries",
		Description:    "Entries without <loc>, invalid <lastmod> values, a wrong namespace or nested sitemap indexes were found.",
		Recommendation: "Fix the listed entries. lastmod must be a W3C Datetime and sitemap indexes must not reference other indexes.",
	})
	SitemapOutOfScope = register(Rule{
		ID: "SEO-SITEMAP-006", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Sitemap lists URLs outside the site",
		Description:    "Sitemap URLs must belong to the host the sitemap describes unless cross-submission is verified.",
		Recommendation: "List only URLs of this site, or verify cross-site submission in the search engines' webmaster tools.",
	})
	SitemapInvalidLoc = register(Rule{
		ID: "SEO-SITEMAP-007", Category: CategoryCrawlability, Severity: SeverityLow,
		Title:          "Sitemap contains invalid URLs",
		Description:    "Some <loc> values are relative, use unsupported schemes or cannot be parsed.",
		Recommendation: "Use fully qualified, properly escaped http or https URLs in <loc>.",
	})
	SitemapLimitReached = register(Rule{
		ID: "SEO-SITEMAP-008", Category: CategoryCrawlability, Severity: SeverityInfo,
		Title:          "Sitemap discovery stopped at a CrawlGrade limit",
		Description:    "More sitemap files or URLs exist than CrawlGrade reads; sitemap comparisons cover only the part that was read.",
		Recommendation: "No action needed for the site. Audit large sitemaps in parts if a complete comparison is required.",
	})
)
