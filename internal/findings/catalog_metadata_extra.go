package findings

// Metadata: hreflang (regional targeting).
var (
	HreflangMissing = register(Rule{
		ID: "SEO-HREFLANG-001", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Page has no hreflang alternates",
		Description:    "A page with regional or language versions should declare them with rel=alternate hreflang links (and an x-default when the site serves several languages). Without them, search engines guess the intended audience.",
		Recommendation: "Add a rel=alternate hreflang link for every language/region version, including an x-default that points to the fallback page.",
	})
	HreflangConflict = register(Rule{
		ID: "SEO-HREFLANG-002", Category: CategoryMetadata, Severity: SeverityMedium,
		Title:          "hreflang alternates are inconsistent",
		Description:    "Every hreflang link should list all the other versions, including a self-reference and an x-default, and each version should point back to the others. A version that is missing from the list, or a self-reference that is absent, makes the network unreliable.",
		Recommendation: "Give every language version the same complete set of hreflang links, each pointing back to all the others.",
	})
	HreflangInvalid = register(Rule{
		ID: "SEO-HREFLANG-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "hreflang value is not a valid language or region",
		Description:    "hreflang values must be ISO 639-1 language codes, optionally followed by a hyphen and an ISO 3166-1 region code (for example pl, pl-PL, en-US). Unknown values are ignored by search engines.",
		Recommendation: "Use valid language and region codes, for example pl, pl-PL or en-UK.",
	})
	HreflangSelfMissing = register(Rule{
		ID: "SEO-HREFLANG-004", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "hreflang set has no self-reference",
		Description:    "A page that declares hreflang alternates should also link to itself. Search engines use the self-reference to confirm the page is one of the alternates.",
		Recommendation: "Add a rel=alternate hreflang link that points back to this page.",
	})
)

// Metadata: social preview metadata (Open Graph and Twitter Card).
var (
	SocialMissing = register(Rule{
		ID: "SEO-SOCIAL-001", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Page has no social preview metadata",
		Description:    "No Open Graph or Twitter Card meta tags were found. When the page is shared on social networks it is previewed without a title, description or image, which makes the share less recognizable.",
		Recommendation: "Add og:title, og:description and og:image (and the corresponding twitter: tags) so shared links preview well.",
	})
	SocialInconsistent = register(Rule{
		ID: "SEO-SOCIAL-002", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Open Graph and Twitter Card metadata disagree",
		Description:    "The page sets both an Open Graph and a Twitter Card meta tag for the same property but the values differ. Most networks read the Open Graph tags, so the Twitter values are usually ignored.",
		Recommendation: "Keep the Open Graph and Twitter Card values in agreement, or drop the redundant Twitter tags.",
	})
	SocialImageMissing = register(Rule{
		ID: "SEO-SOCIAL-003", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Social preview image is missing or too small",
		Description:    "An og:image (or twitter:image) that is absent, relative, or narrower than about 120 pixels or shorter than about 120 pixels is often dropped or shown poorly by social networks.",
		Recommendation: "Share an absolute image at least 120x120 pixels (1200x630 is recommended) that is not blocked by robots.txt.",
	})
	SocialImageInvalid = register(Rule{
		ID: "SEO-SOCIAL-004", Category: CategoryMetadata, Severity: SeverityLow,
		Title:          "Social preview image URL is invalid",
		Description:    "The og:image or twitter:image value cannot be parsed or uses a scheme other than http or https.",
		Recommendation: "Point the image tag at an absolute http(s) URL.",
	})
)

// Metadata: anchor text of internal links.
var (
	AnchorEmpty = register(Rule{
		ID: "SEO-ANCHOR-001", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Internal link has empty anchor text",
		Description:    "An internal link carries no visible text (for example an image without alt text or an empty <a>). Crawlers and screen readers then have no description of the target.",
		Recommendation: "Give internal links descriptive text or alt text.",
	})
	AnchorDuplicate = register(Rule{
		ID: "SEO-ANCHOR-002", Category: CategoryMetadata, Severity: SeverityInfo,
		Title:          "Internal link anchor text is reused",
		Description:    "The same anchor text is used for links to different pages. Ambiguous text gives crawlers and screen readers little to distinguish the targets.",
		Recommendation: "Use anchor text that describes each link's target rather than a shared word like \"here\" or \"click here\".",
	})
)
