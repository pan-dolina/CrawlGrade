# Development plan

CrawlGrade is built incrementally; each step lands as one or more reviewed
commits with tests.

## Milestone 1 - Safe foundations

- [ ] CLI root and version commands, exit codes
- [ ] Finding and severity model with a documented rule catalog
- [ ] Bounded HTTP fetcher (timeouts, body and decompression limits, manual redirects)
- [ ] SSRF policy: guarded dialer, redirect target checks, DNS rebinding mitigation
- [ ] URL normalization and scope control, crawl trap heuristics
- [ ] Bounded crawler with worker pool and rate limiting

## Milestone 2 - Crawl inputs

- [ ] robots.txt (RFC 9309)
- [ ] XML sitemaps, sitemap indexes, gzip with decompression limits
- [ ] Controlled local test site

## Milestone 3 - Page analysis

- [ ] Title, description, headings, canonical, robots meta and X-Robots-Tag
- [x] JSON-LD structured data and breadcrumbs (JSON-LD and microdata)
- [x] hreflang
- [x] Images, OpenGraph, Twitter cards
- [x] Internal link graph

## Milestone 4 - Content analysis

- [ ] Main content extraction and boilerplate removal
- [ ] Polish and English tokenization with static stopword lists
- [ ] Unigram, bigram and trigram term strength, per page and site-wide
- [ ] Exact and near-duplicate detection

## Milestone 5 - Reporting

- [ ] Passive web hygiene checks
- [ ] Terminal, versioned JSON and standalone escaped HTML reports
- [ ] Baseline output and diff

## Milestone 6 - Assurance and release

- [ ] Functional and golden tests against the compiled binary
- [ ] Fuzz targets for every parser, benchmarks
- [ ] CI on Linux, macOS and Windows; static analysis and vulnerability scanning
- [ ] SBOM, reproducible builds, keyless signing and provenance
- [ ] Final security review, documentation, v0.1.0
