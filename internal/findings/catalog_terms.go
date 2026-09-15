package findings

// Term strength: how strongly n-grams are associated with the site and pages.
var (
	TermCommon = register(Rule{
		ID: "SEO-TERMS-001", Category: CategoryContent, Severity: SeverityInfo,
		Title:          "Site-wide term is spread thin",
		Description:    "A term appears on many pages of the site but never dominates any single page. This is expected for terms that describe the whole site rather than one page.",
		Recommendation: "No action needed; this is informational. Terms that dominate one page are reported separately as page-specific.",
	})
	TermPageSpecific = register(Rule{
		ID: "SEO-TERMS-002", Category: CategoryContent, Severity: SeverityInfo,
		Title:          "Page-specific term",
		Description:    "A term is concentrated on a single page and is a good candidate for describing that page's topic to search engines.",
		Recommendation: "No action needed; this is informational. Consider whether the page title and headings reflect these terms.",
	})
	TermDuplicate = register(Rule{
		ID: "SEO-TERMS-003", Category: CategoryContent, Severity: SeverityLow,
		Title:          "Identical term profile on duplicate pages",
		Description:    "Two or more indexable pages share the same dominant n-grams, which reinforces that they are near-duplicates competing for the same topic.",
		Recommendation: "Differentiate the pages, or point duplicates at one canonical URL with rel=canonical.",
	})
)
