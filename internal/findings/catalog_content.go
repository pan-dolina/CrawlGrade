package findings

// Content: main content extraction and length.
var (
	ContentBoilerplate = register(Rule{
		ID: "SEO-CONTENT-001", Category: CategoryContent, Severity: SeverityInfo,
		Title:          "Page is mostly boilerplate",
		Description:    "After removing navigation, headers, footers and other shared chrome, the main content is a small fraction of the page. A page that is mostly boilerplate carries little topical signal of its own.",
		Recommendation: "Give the page its own substantial text so the main content is not dominated by shared site chrome.",
	})
	ContentVeryShort = register(Rule{
		ID: "SEO-CONTENT-002", Category: CategoryContent, Severity: SeverityLow,
		Title:          "Main content is very short",
		Description:    "The extracted main content has very few tokens. Thin pages often describe little and can be hard to place in a topic.",
		Recommendation: "Add descriptive, unique text about the page topic where it adds value for the visitor.",
	})
	ContentNoText = register(Rule{
		ID: "SEO-CONTENT-003", Category: CategoryContent, Severity: SeverityMedium,
		Title:          "Main content has no extractable text",
		Description:    "After removing boilerplate, the page has no tokens left. This usually means the visible text lives in a place the extractor does not treat as main content (for example inside navigation or a script block).",
		Recommendation: "Ensure the page's main text is in the primary content region and not inside <nav>, <script>, <style> or similar.",
	})
)
