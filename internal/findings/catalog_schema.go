package findings

// Structured data: JSON-LD and microdata.
var (
	SchemaInvalid = register(Rule{
		ID: "SEO-SCHEMA-001", Category: CategoryStructuredData, Severity: SeverityMedium,
		Title:          "JSON-LD block is not valid JSON",
		Description:    "A script of type application/ld+json could not be parsed. Search engines ignore the whole block.",
		Recommendation: "Fix the JSON syntax (trailing commas, unquoted keys, unescaped quotes) and validate the markup.",
	})
	SchemaUntyped = register(Rule{
		ID: "SEO-SCHEMA-002", Category: CategoryStructuredData, Severity: SeverityLow,
		Title:          "JSON-LD item has no @type",
		Description:    "A top-level JSON-LD object has no @type, so it does not describe a recognizable entity.",
		Recommendation: "Add an @type such as Organization, Product or Article.",
	})
	SchemaContext = register(Rule{
		ID: "SEO-SCHEMA-003", Category: CategoryStructuredData, Severity: SeverityLow,
		Title:          "JSON-LD @context is missing or not schema.org",
		Description:    "Without @context set to https://schema.org, type and property names are not interpreted as schema.org vocabulary.",
		Recommendation: "Set \"@context\": \"https://schema.org\" on every top-level object or @graph.",
	})
	SchemaRequired = register(Rule{
		ID: "SEO-SCHEMA-004", Category: CategoryStructuredData, Severity: SeverityMedium,
		Title:          "Structured data item lacks key properties",
		Description:    "A recognized type is missing properties CrawlGrade considers essential (for example Product without name, or FAQPage questions without answers). This is a basic check, not a rich result validation.",
		Recommendation: "Add the listed properties; check the search engine documentation for the full requirements of each feature.",
	})
	SchemaRecommended = register(Rule{
		ID: "SEO-SCHEMA-005", Category: CategoryStructuredData, Severity: SeverityInfo,
		Title:          "Structured data item lacks recommended properties",
		Description:    "Recommended properties such as Article author or Organization logo are missing.",
		Recommendation: "Add the listed properties where the information exists on the page.",
	})
	SchemaInvalidURL = register(Rule{
		ID: "SEO-SCHEMA-006", Category: CategoryStructuredData, Severity: SeverityLow,
		Title:          "Structured data contains invalid URLs",
		Description:    "url, logo, image, sameAs or item values use unsupported schemes or cannot be parsed.",
		Recommendation: "Use absolute http(s) URLs in structured data.",
	})
	SchemaLimits = register(Rule{
		ID: "SEO-SCHEMA-007", Category: CategoryStructuredData, Severity: SeverityInfo,
		Title:          "Structured data exceeds CrawlGrade limits",
		Description:    "The page has more than 50 JSON-LD blocks, a block larger than 1 MiB, or deeper nesting than CrawlGrade analyses. Only part of it was checked.",
		Recommendation: "Review whether this much structured data is intended; very large blocks slow down pages.",
	})
	SchemaStartMissing = register(Rule{
		ID: "SEO-SCHEMA-008", Category: CategoryStructuredData, Severity: SeverityInfo,
		Title:          "Start page has no site-level structured data",
		Description:    "No Organization, LocalBusiness or WebSite item was found on the start page.",
		Recommendation: "Describe the organization or website with JSON-LD on the home page.",
	})
)
