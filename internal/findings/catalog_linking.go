package findings

// Internal linking: the structure of links between the crawled pages.
var (
	LinkOrphan = register(Rule{
		ID: "SEO-LINK-001", Category: CategoryInternalLinking, Severity: SeverityMedium,
		Title:          "Page is not reached by any internal link",
		Description:    "The page was crawled (it is listed in a sitemap or reached directly) but no other crawled page links to it. A page that is not linked is hard for search engines to discover and to judge as important.",
		Recommendation: "Link to the page from a related page, or confirm that it should be excluded from the audit.",
	})
	LinkSelf = register(Rule{
		ID: "SEO-LINK-002", Category: CategoryInternalLinking, Severity: SeverityInfo,
		Title:          "Page links to itself",
		Description:    "A page contains an internal link to its own URL. A self link passes no authority and is usually a template mistake.",
		Recommendation: "Remove the link to the current page.",
	})
	LinkDuplicated = register(Rule{
		ID: "SEO-LINK-003", Category: CategoryInternalLinking, Severity: SeverityInfo,
		Title:          "Page is linked from many pages with the same anchor",
		Description:    "Several pages link to the same target using identical anchor text. Repeated anchors dilute the signal that anchor text gives about the target's topic.",
		Recommendation: "Vary the anchor text so it reflects each linking page's context.",
	})
	LinkBroken = register(Rule{
		ID: "SEO-LINK-004", Category: CategoryInternalLinking, Severity: SeverityHigh,
		Title:          "Internal link points to a page that could not be fetched",
		Description:    "An internal link resolves to a URL on this site that returned a 4xx/5xx status, was blocked by the network policy, or was not reached within the crawl limits.",
		Recommendation: "Fix the link or remove it; a broken internal link wastes crawl budget and frustrates users.",
	})
	LinkExternal = register(Rule{
		ID: "SEO-LINK-005", Category: CategoryInternalLinking, Severity: SeverityInfo,
		Title:          "Internal link points outside the site",
		Description:    "A link leaves the audited host. External links are recorded but not crawled; mark commercial or sponsored links with rel=\"sponsored\" or rel=\"nofollow\".",
		Recommendation: "No action needed for editorial links. Add rel=\"sponsored\" or rel=\"nofollow\" to paid or sponsored links.",
	})
	LinkResource = register(Rule{
		ID: "SEO-LINK-006", Category: CategoryInternalLinking, Severity: SeverityInfo,
		Title:          "Internal link points to a non-HTML resource",
		Description:    "A link targets a file such as a PDF or image rather than another page. This is fine for genuine downloads, but linking to a resource where a page was intended wastes crawl budget.",
		Recommendation: "Link to the page that hosts the resource, or keep the link only where the file is the intended destination.",
	})
	LinkNoopener = register(Rule{
		ID: "SEO-LINK-007", Category: CategoryInternalLinking, Severity: SeverityInfo,
		Title:          "Internal link opens in a new tab without rel=\"noopener\"",
		Description:    "A link with target=\"_blank\" that omits rel=\"noopener\" (or rel=\"noreferrer\") lets the new page control the opener window via window.opener, a privacy and safety risk.",
		Recommendation: "Add rel=\"noopener\" (or rel=\"noreferrer\") to every link that opens a new tab.",
	})
)
