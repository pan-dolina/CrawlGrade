package findings

var (
	ImageAlt        = register(Rule{ID: "SEO-IMAGE-001", Category: CategoryContent, Severity: SeverityLow, Title: "Image has no alt attribute", Description: "The image lacks a text alternative.", Recommendation: "Describe informative images; use alt=\"\" for decorative images."})
	ImageDimensions = register(Rule{ID: "SEO-IMAGE-002", Category: CategoryContent, Severity: SeverityInfo, Title: "Image has no positive width and height", Description: "Intrinsic dimensions help reserve layout space.", Recommendation: "Declare width and height consistent with the image aspect ratio."})
	ImageLoading    = register(Rule{ID: "SEO-IMAGE-003", Category: CategoryContent, Severity: SeverityLow, Title: "Image loading value is invalid", Description: "The loading attribute accepts lazy or eager.", Recommendation: "Use lazy for suitable offscreen images and eager for critical visible images."})
	SocialRequired  = register(Rule{ID: "SEO-SOCIAL-005", Category: CategoryMetadata, Severity: SeverityLow, Title: "Social metadata is incomplete", Description: "The declared preview lacks basic properties.", Recommendation: "Complete the listed properties for the declared preview format."})
)
