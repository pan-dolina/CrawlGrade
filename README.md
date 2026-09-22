# CrawlGrade

CrawlGrade is a local command-line tool for technical SEO audits. It crawls a
website within strict safety limits and reports on crawlability, metadata,
content, structured data, internal linking and passive web hygiene.

CrawlGrade does **not** predict search engine rankings.

Build with the toolchain declared in `go.mod`:

```sh
go build -o bin/crawlgrade ./cmd/crawlgrade
bin/crawlgrade https://example.com --max-pages 100
bin/crawlgrade https://example.com --json --output baseline.json
bin/crawlgrade https://example.com --format html --output report.html
bin/crawlgrade https://example.com --keywords 'technical SEO,badanie wzroku'
bin/crawlgrade diff baseline.json current.json --json
```

Use `--help` for all options. `--max-depth 0` audits only the start page.
`--timeout` bounds each fetch, including redirects; the crawler also has a
ten-minute runtime limit. Requests start at no more than 5 per second by
default, including redirects, robots.txt and sitemaps.

Private networks are blocked by default. For a development site:

```sh
go run ./internal/tools/testsite -addr 127.0.0.1:8080
bin/crawlgrade http://127.0.0.1:8080 --allow-private
```

Cloud metadata addresses remain blocked. Environment proxies, cookies and
automatic HTTP redirects are disabled. Optional `--check-external-links`
makes at most 100 distinct HEAD checks through the same guarded fetcher.

Reports include metadata, hreflang, images, JSON-LD and microdata breadcrumbs,
content duplicates, weighted terms, observed HTTP failures, sitemap comparisons
and internal link counts. HTML output is standalone, escaped and script-free.
JSON uses schema version `"1"`; `--baseline FILE --diff` compares a live audit
with a saved report. Web hygiene is reported and scored separately.

Exit codes: 0 completed; 1 reached `--fail-on`; 2 invalid input; 3 target blocked
by network policy or robots.txt; 4 unavailable start page or interruption;
5 internal/output error. A bounded crawl can finish successfully with skipped
URLs: inspect `summary.stop_reason` and `summary.skipped` before interpreting it
as a complete site inventory.

Limits and interpretation: [crawling](docs/crawling.md),
[analysis and scores](docs/analysis.md), [web hygiene](docs/web-hygiene.md).
Development: [architecture](ARCHITECTURE.md), [journal](docs/development.md),
[plan](DEVELOPMENT_PLAN.md), [release verification](docs/release-verification.md).

```sh
go test ./...
go test -race ./...
go vet ./...
scripts/fuzz.sh 5s
```

Licensed under Apache-2.0. The v0.1.0 release workflow is prepared; publication
and hosted platform validation are separate from local verification.
