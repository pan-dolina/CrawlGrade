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

var (
	WebHygieneRedirect = register(Rule{ID: "WEB-HYGIENE-004", Category: CategoryWebHygiene, Severity: SeverityLow, Title: "HTTP to HTTPS redirect was not confirmed", Description: "The HTTP variant did not successfully redirect to HTTPS, or its response was unavailable.", Recommendation: "Verify that the HTTP variant redirects to the intended HTTPS URL."})
	WebHygieneMixed    = register(Rule{ID: "WEB-HYGIENE-005", Category: CategoryWebHygiene, Severity: SeverityMedium, Title: "HTTPS page references HTTP resources", Description: "Some embedded resources use plain HTTP; browsers may upgrade or block them.", Recommendation: "Use HTTPS URLs for embedded resources."})
	WebHygieneHSTS     = register(Rule{ID: "WEB-HYGIENE-006", Category: CategoryWebHygiene, Severity: SeverityLow, Title: "Missing Strict-Transport-Security (HSTS) header", Description: "The HTTPS response does not send Strict-Transport-Security, so browsers cannot enforce HTTPS for future visits.", Recommendation: "Send Strict-Transport-Security with a max-age of at least 31536000 (1 year)."})
	WebHygieneCSP      = register(Rule{ID: "WEB-HYGIENE-007", Category: CategoryWebHygiene, Severity: SeverityLow, Title: "Missing Content-Security-Policy (CSP) header", Description: "The response does not send a Content-Security-Policy header, reducing defense-in-depth against content injection.", Recommendation: "Deploy a Content-Security-Policy appropriate for the application."})
	WebHygieneFrame    = register(Rule{ID: "WEB-HYGIENE-008", Category: CategoryWebHygiene, Severity: SeverityLow, Title: "Missing clickjacking protection (X-Frame-Options / frame-ancestors)", Description: "The response does not send X-Frame-Options or CSP frame-ancestors, leaving the page potentially frameable by third parties.", Recommendation: "Send X-Frame-Options (DENY or SAMEORIGIN) or CSP frame-ancestors to prevent unauthorized framing."})
)
