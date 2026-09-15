# Architecture

CrawlGrade is a single static Go binary. The code is split into packages
with one responsibility each; network access is confined to `fetcher`, and
everything that interprets site content is a pure function of fetched data,
which makes it testable without a network.

```
cmd/crawlgrade          main: signal handling, exit code
internal/
  cli/                  cobra commands, flag validation, exit codes
  audit/                orchestration: crawl -> analyses -> findings -> scores
  netguard/             IP classification and guarded dialer (SSRF policy)
  fetcher/              bounded HTTP client: redirects, size and decompression limits
  urlnorm/              URL normalization, scope, trap heuristics
  crawler/              level-synchronous BFS, worker pool, rate limiting
  robots/               robots.txt parser and matcher (RFC 9309)
  sitemap/              XML sitemap / sitemap index / .xml.gz parsing
  htmlcheck/            HTML parsing into a page model; per-page metadata checks
  structureddata/       JSON-LD extraction, type recognition, breadcrumbs, microdata
  links/                internal link graph, inbound/outbound metrics
  content/              main content extraction and boilerplate removal
  terms/                tokenization, stopwords, n-grams, term strength
  duplicates/           exact hashes and SimHash near-duplicate detection
  webhygiene/           passive security header and mixed content checks
  findings/             finding model, severities, rule catalog
  report/               text, JSON and HTML renderers; baseline diff
  testsite/             deterministic local site used by functional tests
  version/              build metadata
test/
  functional/           runs the compiled binary against testsite
  smoke/                checks a release binary (build tag "smoke")
```

## Data flow

```
URL ─► cli ─► audit.Run
                 │
                 ├─► robots.Fetch (per host)
                 ├─► sitemap.Fetch (robots Sitemap: lines + /sitemap.xml)
                 ├─► crawler.Crawl ─► fetcher.Get ─► netguard dialer
                 │        └─► htmlcheck.Parse ─► Page model (links, meta, text)
                 ├─► site-level analyses on []Page:
                 │     links graph, duplicate titles/descriptions, canonical
                 │     targets, hreflang reciprocity, sitemap comparison,
                 │     content + terms, duplicates, web hygiene
                 └─► findings + scores ─► report.{Text,JSON,HTML}
```

## Principles

- **Untrusted input everywhere.** Every loop and allocation driven by site
  content has an explicit bound. Parsers never panic on malformed input;
  fuzz targets back this up.
- **Deterministic output.** Given the same site, the same pages are crawled
  regardless of concurrency (the crawler processes one depth level at a
  time and schedules URLs in sorted order), and every list in a report is
  sorted. This keeps golden tests and baseline diffs stable.
- **Stable identifiers.** Finding IDs and the JSON `schema_version` are
  public interface; IDs are never renumbered or reused.
- **Separation of concerns.** Analysis packages return data and findings and
  never print. Renderers never make network requests.
- **Minimal dependencies.** Standard library first. See
  [docs/development.md](docs/development.md) for every dependency decision.
