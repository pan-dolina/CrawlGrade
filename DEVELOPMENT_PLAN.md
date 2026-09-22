# Development plan

CrawlGrade is built incrementally; each step lands as one or more reviewed
commits with tests.

## Milestone 1 - Safe foundations (done)

- [x] CLI root and version commands, exit codes
- [x] Finding and severity model with a documented rule catalog
- [x] Bounded HTTP fetcher (timeouts, body and decompression limits, manual redirects)
- [x] SSRF policy: guarded dialer, redirect target checks, DNS rebinding mitigation
- [x] URL normalization and scope control, crawl trap heuristics
- [x] Bounded crawler with worker pool and rate limiting

## Milestone 2 - Crawl inputs (done)

- [x] robots.txt (RFC 9309)
- [x] XML sitemaps, sitemap indexes, gzip with decompression limits
- [x] Controlled local test site

## Milestone 3 - Page analysis (implemented)

- [x] Title, description, headings, canonical, robots meta and X-Robots-Tag
- [x] JSON-LD structured data and breadcrumbs (JSON-LD and microdata)
- [x] hreflang: language/region validation, duplicates, x-default, reciprocity, relation to html lang
- [x] Images: alt, width/height, loading, broken URLs
- [x] OpenGraph and Twitter cards: basic declared-property and image-URL checks
- [x] Internal link graph with inbound/outbound metrics, rel attributes and observed orphan pages; low inbound counts expose weakly linked pages

## Milestone 4 - Content analysis (implemented)

- [x] Main/article content extraction with documented boilerplate and hidden-node removal heuristics
- [x] Polish and English tokenization with static stopword lists in the repository
- [x] Weighted unigram, bigram and trigram term strength (TF-IDF, 0-100), per page and site-wide
- [x] Exact and near-duplicate detection connected to the audit

## Milestone 5 - Reporting (implemented)

- [x] Passive web hygiene: HTTPS, nosniff, Referrer-Policy, HTTP redirect observation and mixed-resource markup
- [x] Terminal, versioned JSON and HTML reports with IDs, terms, link metrics, separate scores and HTML CSP
- [x] Baseline and diff (`--baseline`/`--diff`, offline `crawlgrade diff`, aggregate deltas)
- [x] Audit orchestration: analyses run before body disposal; HTTP and sitemap comparisons use observed results
- [x] CLI flags from the specification (`--json`, `--output`, `--keywords`, `--timeout`, `--user-agent`, `--requests-per-second`, `--check-external-links`, `--allow-private`)

A review on 2026-09-15 found that milestones 3-5 had been marked complete
while the items above were missing or only partly implemented. The implementation work on 2026-09-22 addresses those gaps; analysis
heuristics and limits are documented in `docs/analysis.md`.

## Milestone 6 - Assurance and release

- [x] Functional and golden tests against the compiled binary
- [x] Fuzz targets for HTML/metadata, JSON-LD/microdata, robots, sitemap, URL, content, terms and saved report parsers; extraction benchmark
- [x] CI definitions for Linux, macOS and Windows; static analysis and vulnerability scanning
- [x] SBOM generation and reproducible build scripts
- [x] Release workflow for keyless signing, provenance and platform smoke tests
- [x] Local security review and user/developer documentation
- [ ] Hosted CI/release execution and publication of v0.1.0 (requires GitHub)

Local source, workflow definitions and release preparation are implemented.
A workflow definition is not a successful hosted run; published signatures,
provenance and platform smoke results remain release-time verification.
