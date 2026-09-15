package findings

// Duplicate content: exact and near-duplicate pages.
var (
	DuplicateExact = register(Rule{
		ID: "SEO-DUPLICATE-001", Category: CategoryContent, Severity: SeverityMedium,
		Title:          "Exact duplicate pages",
		Description:    "Two or more indexable pages have identical extracted main content. Exact duplicates compete for the same topic and waste crawl budget.",
		Recommendation: "Keep one page and point the others at it with rel=canonical, or remove the redundant pages.",
	})
	DuplicateNear = register(Rule{
		ID: "SEO-DUPLICATE-002", Category: CategoryContent, Severity: SeverityLow,
		Title:          "Near-duplicate pages",
		Description:    "Two or more indexable pages have very similar extracted main content (they differ only in a few words). Search engines may treat them as the same page.",
		Recommendation: "Differentiate the pages' text, or point duplicates at one canonical URL with rel=canonical.",
	})
)
