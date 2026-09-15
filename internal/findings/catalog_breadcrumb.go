package findings

// Structured data: breadcrumbs.
var (
	BreadcrumbMalformed = register(Rule{
		ID: "SEO-BREADCRUMB-001", Category: CategoryStructuredData, Severity: SeverityMedium,
		Title:          "Breadcrumb markup is malformed",
		Description:    "A BreadcrumbList has no itemListElement, or its elements are not ListItem objects (JSON-LD) or ListItem itemscopes (microdata).",
		Recommendation: "Structure breadcrumbs as a BreadcrumbList whose itemListElement entries are ListItem items.",
	})
	BreadcrumbInvalid = register(Rule{
		ID: "SEO-BREADCRUMB-002", Category: CategoryStructuredData, Severity: SeverityLow,
		Title:          "Breadcrumb entries are incomplete",
		Description:    "Breadcrumb entries lack a name, an item URL (required for all but the last entry) or a positive integer position, or positions do not start at 1.",
		Recommendation: "Give every ListItem a name, a position starting at 1 and an item URL (optional only for the current page).",
	})
	BreadcrumbDuplicate = register(Rule{
		ID: "SEO-BREADCRUMB-003", Category: CategoryStructuredData, Severity: SeverityLow,
		Title:          "Breadcrumb entries are duplicated",
		Description:    "Several entries of one BreadcrumbList share a position or an item URL.",
		Recommendation: "Use each position and URL once per breadcrumb trail.",
	})
)
