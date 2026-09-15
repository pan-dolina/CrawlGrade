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

## Milestone 3 - Page analysis (in progress)

- [x] Title, description, headings, canonical, robots meta and X-Robots-Tag
- [x] JSON-LD structured data and breadcrumbs (JSON-LD and microdata)
- [ ] hreflang: language/region validation, duplicates, x-default, reciprocity, relation to html lang
- [ ] Images: alt, width/height, loading, broken URLs
- [~] OpenGraph and Twitter cards (parsed; required property checks missing)
- [ ] Internal link graph with inbound/outbound metrics, rel attributes, weakly linked and orphan-like pages

## Milestone 4 - Content analysis (in progress)

- [ ] Main content extraction with documented boilerplate removal (current version only skips chrome elements)
- [ ] Polish and English tokenization with static stopword lists in the repository
- [ ] Weighted unigram, bigram and trigram term strength (TF-IDF, 0-100), per page and site-wide
- [~] Exact and near-duplicate detection (package exists, not wired into the audit)

## Milestone 5 - Reporting (in progress)

- [~] Passive web hygiene checks (HTTPS, nosniff, Referrer-Policy only)
- [~] Terminal, versioned JSON and HTML reports (no finding IDs, terms, link graph or scores; no CSP)
- [~] Baseline and diff (`--baseline`/`--diff`; `crawlgrade diff` and aggregate deltas missing)
- [~] Audit orchestration (structured data, duplicates, HTTP and sitemap comparisons not wired; page bodies are dropped before content analysis)
- [ ] CLI flags from the specification (`--json`, `--output`, `--keywords`, `--timeout`, `--user-agent`, `--requests-per-second`, `--check-external-links`, `--allow-private`)

A review on 2026-09-15 found that milestones 3-5 had been marked complete
while the items above were missing or only partly implemented. The status
now reflects the code.

## Milestone 6 - Assurance and release

- [ ] Functional and golden tests against the compiled binary
- [ ] Fuzz targets for every parser, benchmarks
- [ ] CI on Linux, macOS and Windows; static analysis and vulnerability scanning
- [ ] SBOM, reproducible builds, keyless signing and provenance
- [ ] Final security review, documentation, v0.1.0
