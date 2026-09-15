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
