# Web hygiene checks

Web hygiene is reported **separately** from the SEO score. It describes a few
security-relevant properties that are visible to any visitor.

All checks are passive: CrawlGrade reads the response headers and markup of
pages it fetched for the SEO audit, plus at most one extra request to the
`http://` variant of the start URL to test the HTTP to HTTPS redirect. No
payloads, probes or unusual requests are sent.

This is not a vulnerability scan and a good result does not mean a site is
secure.
