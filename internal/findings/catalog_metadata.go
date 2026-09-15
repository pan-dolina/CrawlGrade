package findings

// Metadata: titles.
var (
	TitleMissing = register(Rule{
		ID: "SEO-TITLE-001", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Page has no title",
		Description:    "The document has no <title> element. Search engines then generate a title from page content.",
		Recommendation: "Add a unique, descriptive <title> in the document head.",
	})
	TitleEmpty = register(Rule{
		ID: "SEO-TITLE-002", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Page title is empty",
		Description:    "The <title> element contains no text.",
		Recommendation: "Write a title that describes the page's main topic.",
	})
	TitleMultiple = register(Rule{
		ID: "SEO-TITLE-003", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Page has multiple title elements",
		Description:    "More than one <title> element was found. Browsers and crawlers use the first one; the others usually come from templates or injected markup.",
		Recommendation: "Keep a single <title> element in the head.",
	})
	TitleDuplicate = register(Rule{
		ID: "SEO-TITLE-004", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Title is shared with other pages",
		Description:    "The same title is used on several indexable pages, which makes them harder to distinguish in search results.",
		Recommendation: "Give each page a title that reflects its specific content.",
	})
	TitleShort = register(Rule{
		ID: "SEO-TITLE-005", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Title is unusually short",
		Description:    "Heuristic: the title has fewer than 15 characters and may not describe the page well.",
		Recommendation: "Consider a more descriptive title. Length is a guideline, not a rule.",
	})
	TitleLong = register(Rule{
		ID: "SEO-TITLE-006", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Title is unusually long",
		Description:    "Heuristic: the title has more than 65 characters. Search engines truncate titles by pixel width, so the end may not be shown.",
		Recommendation: "Put the most important words first. Length is a guideline, not a rule.",
	})
)

// Metadata: meta descriptions.
var (
	DescriptionMissing = register(Rule{
		ID: "SEO-DESC-001", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Page has no meta description",
		Description:    "No <meta name=\"description\"> element was found. Search engines then build snippets from page content.",
		Recommendation: "Add a meta description summarizing the page.",
	})
	DescriptionEmpty = register(Rule{
		ID: "SEO-DESC-002", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Meta description is empty",
		Description:    "The meta description element has no content.",
		Recommendation: "Fill in the description or remove the empty element.",
	})
	DescriptionMultiple = register(Rule{
		ID: "SEO-DESC-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Page has multiple meta descriptions",
		Description:    "More than one meta description element was found; which one is used is undefined.",
		Recommendation: "Keep a single meta description.",
	})
	DescriptionDuplicate = register(Rule{
		ID: "SEO-DESC-004", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Meta description is shared with other pages",
		Description:    "The same meta description is used on several indexable pages.",
		Recommendation: "Write descriptions specific to each page, or omit them rather than repeating one.",
	})
	DescriptionLength = register(Rule{
		ID: "SEO-DESC-005", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Meta description length is outside the usual range",
		Description:    "Heuristic: the description is shorter than about 50 or longer than about 160 characters. Snippet length varies by device and query.",
		Recommendation: "Review the description; length is an approximate guideline only.",
	})
)

// Metadata: headings.
var (
	HeadingNoH1 = register(Rule{
		ID: "SEO-HEADING-001", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Page has no H1 heading",
		Description:    "No <h1> element was found. The main heading helps users and crawlers understand the page topic.",
		Recommendation: "Add one <h1> that states the page's main topic.",
	})
	HeadingMultipleH1 = register(Rule{
		ID: "SEO-HEADING-002", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Page has multiple H1 headings",
		Description:    "Heuristic: several <h1> elements were found. HTML allows this and search engines cope with it, but a single H1 keeps the outline clear. Disable with --allow-multiple-h1.",
		Recommendation: "Consider using one H1 for the main topic and H2-H6 for sections.",
	})
	HeadingEmpty = register(Rule{
		ID: "SEO-HEADING-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Page has empty headings",
		Description:    "Heading elements without text or image alt text were found; they add noise to the document outline and to screen reader navigation.",
		Recommendation: "Remove empty heading elements or give them text; use CSS for spacing.",
	})
	HeadingHierarchy = register(Rule{
		ID: "SEO-HEADING-004", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Heading levels are skipped",
		Description:    "The heading outline skips levels (for example <h1> followed by <h3>) or does not start with <h1>.",
		Recommendation: "Nest headings without gaps so the outline reflects the content structure.",
	})
	HeadingDuplicateH1 = register(Rule{
		ID: "SEO-HEADING-005", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "H1 is shared with other pages",
		Description:    "The same first H1 text (case-insensitive) appears on several indexable pages.",
		Recommendation: "Use headings that distinguish each page's content.",
	})
)

// Metadata: canonical URLs.
var (
	CanonicalMissing = register(Rule{
		ID: "SEO-CANONICAL-001", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Page has no canonical URL",
		Description:    "No rel=canonical link element or HTTP Link header was found. Search engines then choose the canonical URL themselves.",
		Recommendation: "Declare a self-referencing absolute canonical URL on indexable pages.",
	})
	CanonicalMultiple = register(Rule{
		ID: "SEO-CANONICAL-002", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Page declares conflicting canonical URLs",
		Description:    "Several canonical declarations point to different URLs or include invalid ones. Search engines may ignore all of them.",
		Recommendation: "Keep exactly one canonical declaration.",
	})
	CanonicalRelative = register(Rule{
		ID: "SEO-CANONICAL-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Canonical URL is relative",
		Description:    "A relative canonical URL is resolved against the page URL and can silently point to the wrong host or protocol when pages are served under several addresses.",
		Recommendation: "Use absolute canonical URLs including scheme and host.",
	})
	CanonicalInvalid = register(Rule{
		ID: "SEO-CANONICAL-004", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Canonical URL is invalid",
		Description:    "The canonical href is empty, cannot be parsed, or uses a scheme other than http or https.",
		Recommendation: "Point the canonical link to a valid absolute http(s) URL.",
	})
	CanonicalOutsideHead = register(Rule{
		ID: "SEO-CANONICAL-005", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Canonical link is outside the head",
		Description:    "rel=canonical links in the body are ignored by search engines.",
		Recommendation: "Move the canonical link element into <head>.",
	})
	CanonicalCrossDomain = register(Rule{
		ID: "SEO-CANONICAL-006", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Canonical URL points to another site",
		Description:    "The canonical URL is on a different host. This is correct for syndicated content but a mistake when caused by staging hosts or misconfigured templates.",
		Recommendation: "Confirm that the other site is meant to be the canonical source.",
	})
	CanonicalTargetBroken = register(Rule{
		ID: "SEO-CANONICAL-007", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Canonical target is broken",
		Description:    "The canonical URL returns a 4xx or 5xx status or could not be fetched.",
		Recommendation: "Point the canonical to a live URL that returns 200.",
	})
	CanonicalTargetRedirects = register(Rule{
		ID: "SEO-CANONICAL-008", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Canonical target redirects",
		Description:    "The canonical URL redirects. Canonicals should name the final URL directly.",
		Recommendation: "Replace the canonical with the redirect's final destination.",
	})
	CanonicalLoop = register(Rule{
		ID: "SEO-CANONICAL-009", Category: CategoryMetadata, Severity: SeverityHigh,
		Title:          "Canonical URLs form a loop",
		Description:    "Following canonical declarations leads back to an earlier page, so no page is a stable canonical.",
		Recommendation: "Choose one canonical page and make every duplicate point to it; the canonical page should reference itself.",
	})
	CanonicalChain = register(Rule{
		ID: "SEO-CANONICAL-010", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "Canonical URLs form a chain",
		Description:    "The canonical target itself declares a different canonical URL.",
		Recommendation: "Point every page directly to the final canonical URL.",
	})
)
