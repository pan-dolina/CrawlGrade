package findings

// Passive web hygiene: security-relevant properties visible to any visitor.
// Reported separately from the SEO score.
var (
	WebHygieneInsecure = register(Rule{
		ID: "WEB-HYGIENE-001", Category: CategoryWebHygiene, Severity: SeverityMedium,
		Title:          "Page is served over plain HTTP",
		Description:    "The page is served over plain HTTP instead of HTTPS, so the request and its data can be read and altered in transit by anyone on the path between the client and the server.",
		Recommendation: "Serve the whole site over HTTPS and redirect the http:// variant to it.",
	})
	WebHygieneXCTO = register(Rule{
		ID: "WEB-HYGIENE-002", Category: CategoryWebHygiene, Severity: SeverityLow,
		Title:          "Missing X-Content-Type-Options: nosniff",
		Description:    "The response does not send X-Content-Type-Options: nosniff, so a browser may MIME-sniff a response and execute it as something other than its declared content type.",
		Recommendation: "Send X-Content-Type-Options: nosniff on every response.",
	})
	WebHygieneReferrer = register(Rule{
		ID: "WEB-HYGIENE-003", Category: CategoryWebHygiene, Severity: SeverityLow,
		Title:          "Missing Referrer-Policy",
		Description:    "The response does not send a Referrer-Policy, so the browser may send the full URL, including path or query, to other sites as a side effect of following links.",
		Recommendation: "Send a Referrer-Policy such as strict-origin-when-cross-origin.",
	})
)
