package findings

// Metadata: hreflang alternates.
var (
	HreflangNoReturn = register(Rule{
		ID: "SEO-HREFLANG-007", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "hreflang alternate does not link back",
		Description:    "The page lists an alternate that was crawled but does not list this page among its own hreflang alternates. Search engines ignore hreflang annotations that are not reciprocal.",
		Recommendation: "Give every language version the same complete set of hreflang links, each pointing back to all the others.",
	})
	HreflangConflict = register(Rule{
		ID: "SEO-HREFLANG-002", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "hreflang value points to several URLs",
		Description:    "The same language or region value is declared for different URLs, so the intended alternate is ambiguous.",
		Recommendation: "Declare each hreflang value once per page.",
	})
	HreflangInvalid = register(Rule{
		ID: "SEO-HREFLANG-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "hreflang value is not a valid language or region",
		Description:    "hreflang values must be an ISO 639-1 language code, optionally followed by a script and an ISO 3166-1 alpha-2 region (pl, pl-PL, en-GB, zh-Hant-TW), or x-default. Invalid values such as en-UK or en_US are ignored.",
		Recommendation: "Fix the listed values; the region for the United Kingdom is GB.",
	})
	HreflangSelfMissing = register(Rule{
		ID: "SEO-HREFLANG-004", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "hreflang set has no self-reference",
		Description:    "A page that declares hreflang alternates should also list itself.",
		Recommendation: "Add a rel=alternate hreflang link for this page's own language.",
	})
	HreflangNoDefault = register(Rule{
		ID: "SEO-HREFLANG-005", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "hreflang set has no x-default",
		Description:    "The page lists alternates in several languages but no x-default for users whose language matches none of them.",
		Recommendation: "Add hreflang=\"x-default\" pointing to the fallback or language selection page.",
	})
	HreflangTargetProblem = register(Rule{
		ID: "SEO-HREFLANG-006", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "hreflang alternate is broken, redirects or is noindex",
		Description:    "An alternate URL returns an error, redirects or excludes itself from indexing. Alternates must be indexable canonical URLs.",
		Recommendation: "Point hreflang links at the final, indexable URL of each version.",
	})
)

// Metadata: html lang attribute.
var (
	LangMissing = register(Rule{
		ID: "SEO-LANG-001", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "html element has no lang attribute",
		Description:    "Without <html lang>, browsers, screen readers and translation tools must guess the page language.",
		Recommendation: "Declare the page language, for example <html lang=\"pl\">.",
	})
	LangInvalid = register(Rule{
		ID: "SEO-LANG-002", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "html lang is not a valid language tag",
		Description:    "The lang attribute is empty or is not a BCP 47 language tag with a known language and region.",
		Recommendation: "Use a valid tag such as pl, pl-PL or en-GB.",
	})
	LangMismatch = register(Rule{
		ID: "SEO-LANG-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "html lang disagrees with the page's hreflang self-reference",
		Description:    "The language in <html lang> differs from the language this page declares for itself in hreflang.",
		Recommendation: "Make html lang and the self-referencing hreflang value name the same language.",
	})
)

// HreflangMissing retains the original public rule contract.
var HreflangMissing = register(Rule{ID: "SEO-HREFLANG-001", Category: CategoryMetadata, Severity: SeverityLow, Title: "Page has no hreflang alternates", Description: "A page with known language versions has no alternates.", Recommendation: "Declare alternate language versions when available."})
