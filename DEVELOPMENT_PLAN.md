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

## Milestone 3 - Page analysis (done)

- [x] Title, description, headings, canonical, robots meta and X-Robots-Tag
- [x] JSON-LD structured data and breadcrumbs (JSON-LD and microdata)
- [x] hreflang
- [x] Images, OpenGraph, Twitter cards
- [x] Internal link graph

## Milestone 4 - Content analysis (done)

- [x] Main content extraction and boilerplate removal
- [x] Polish and English tokenization with static stopword lists
- [x] Unigram, bigram and trigram term strength, per page and site-wide
- [x] Exact and near-duplicate detection

## Milestone 5 - Reporting (done)

- [x] Passive web hygiene checks (`internal/webhygiene`)
- [x] Terminal, versioned JSON and standalone escaped HTML reports (`internal/report`)
- [x] Baseline output and diff (`--baseline`, `--diff`)
- [x] Audit orchestration wiring crawl + analyses + report (`internal/audit`)
- [x] CLI: `--format`, `--fail-on`, `--baseline`, `--diff`, `--no-color`, `GR_ALLOW_PRIVATE`

## Milestone 6 - Assurance and release

- [ ] Functional and golden tests against the compiled binary
- [ ] Fuzz targets for every parser, benchmarks
- [ ] CI on Linux, macOS and Windows; static analysis and vulnerability scanning
- [ ] SBOM, reproducible builds, keyless signing and provenance
- [ ] Final security review, documentation, v0.1.0
