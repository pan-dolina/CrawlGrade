# Development journal

A running record of problems, decisions and changes made while building
CrawlGrade. Newest entries at the bottom of each section.

## Setup

- Toolchain: Go 1.27 locally; `go.mod` declares `go 1.26.0` so the previous
  stable release can still build the project.
- Module path: `github.com/pan-dolina/crawlgrade`.

## Dependency decisions

- **github.com/spf13/cobra** (+ spf13/pflag). The CLI has a root command that
  takes a URL plus `diff` and `version` subcommands. Cobra gives POSIX flags,
  consistent help and argument validation; it has no transitive runtime
  dependencies beyond pflag (mousetrap is Windows-only). The same library is
  used by MailAuthProbe, so the project shares conventions and review effort.

## Design decisions

- Exit codes are defined once in `internal/cli/exitcode.go` and printed in
  `--help`. Cobra's own flag and argument errors are mapped to exit code 2.
- The fetcher never lets `net/http` follow redirects (`CheckRedirect`
  returns `http.ErrUseLastResponse`). Each hop is issued by `Fetch`, so the
  chain is recorded for the report, loops are detected by URL rather than by
  a hop counter alone, and every hop goes through the guarded dialer.
- Transport-level decompression is disabled. The fetcher asks for `gzip`
  only and decodes it itself through two limits: bytes read from the
  connection and bytes produced by the decoder. Stacked encodings
  (`gzip, gzip`) and encodings that were not requested (`br`, `deflate`) are
  rejected instead of being passed to the HTML parser as binary noise.
- A body that exceeds a limit is an error, not a truncated page. Parsing a
  truncated HTML document would produce misleading findings (a missing
  `</html>` is harmless, but so would be a missing canonical that happened to
  sit after the cut).

## SSRF design

- The policy lives in `netguard.Dialer` (ADR 0001). Checking URLs before a
  request cannot see redirects handled inside `net/http`, nor the address a
  name resolves to at connection time.
- **DNS rebinding.** The dialer resolves a name once, checks every returned
  address and dials the checked IP literal. A second resolution never
  happens between check and connect. When an answer mixes public and private
  addresses the host is rejected entirely; dropping only the bad records
  would still let an attacker influence which address is used.
- The `net.Dialer.Control` hook re-checks the socket address immediately
  before `connect(2)`, covering any path that reaches the dialer with a
  literal address.
- Addresses that embed IPv4 (IPv4-mapped, NAT64 `64:ff9b::/96`, 6to4) are
  classified by the embedded address; Teredo is rejected because its
  embedded client address is obfuscated. IPv6 outside `2000::/3` is treated
  as reserved.
- `--allow-private` is for auditing local development sites. Cloud metadata
  addresses stay blocked even then: a local target has no reason to redirect
  to them, and they are the most valuable SSRF target.
- Environment proxies are ignored (`Transport.Proxy = nil`); a proxy would
  make the connection on our behalf to an address we never checked.

## URL normalization

- Normalization follows RFC 3986 sections 6.2.2 and 6.2.3 only: lower-case
  scheme and host, IDNA to ASCII (non-transitional, BiDi rule), default port
  removal, percent-encoding case, decoding of encoded unreserved characters
  and dot-segment removal. Paths are case-sensitive and `/a` and `/a/` are
  different resources on many servers, so neither is changed. Query parameter
  order is preserved; sorting it would merge URLs a site treats differently.
- Percent-encoding is normalized **before** dot segments are removed, so
  `/a/%2e%2e/b` resolves like `/a/../b`.
- Reports show host names in their ASCII (punycode) form. Unicode host names
  can be visually confusable and may contain BiDi characters; ASCII output
  cannot spoof another host in a terminal or an HTML report.
- Percent-encoded host names are rejected: parsers disagree on whether to
  decode them, and the crawler must know exactly which host it contacts.
- Site scope ignores a leading `www.` and the scheme, and includes a
  non-default port. `http://example.com` redirecting to
  `https://www.example.com` is therefore an internal redirect.

## Crawler limits

- The crawl is level-synchronous (ADR 0002). An early version with a shared
  queue produced different page sets for `--concurrency 1` and `8` under
  `--max-pages`; `TestCrawlBreadthFirstAndDeterministic` now pins the order.
- URL admission (seen set, robots, trap guard, queue limit) runs on the
  coordinating goroutine between levels, so the trap guard needs no locking
  and its decisions do not depend on which worker finished first.
- Response bodies are dropped as soon as `Process` has analysed a page.
  Memory therefore scales with the page models, not with raw HTML.
- URLs whose extension names a non-HTML file (`.pdf`, images, archives, ...)
  are checked with `HEAD`: their status matters for broken-link detection
  but their bodies are never needed. They count towards `--max-pages`, which
  bounds the total number of requests rather than only HTML pages.
- `--max-depth 0` is meaningful (audit only the start URL), so unlike the
  other limits a zero depth is not replaced by the default.
- The rate limiter spaces request starts evenly (`1/rps`) and is consulted
  before every redirect hop, not only before each page, so redirect chains
  cannot bypass `--requests-per-second`.

## robots.txt

- RFC 9309 semantics: groups for the same product token are merged; the
  `*` groups apply only when no group names `crawlgrade`; the longest
  matching pattern wins and `Allow` wins ties; `/robots.txt` is always
  allowed.
- 4xx (including 401 and 403) means "no restrictions" and produces only an
  informational finding. 5xx, network errors, redirect loops and responses
  blocked by the network policy mean "disallow everything", as the RFC
  requires. This is deliberately strict: a crawler that ignores a failing
  robots.txt could crawl a site against its operator's wishes.
- Wildcard matching uses the iterative star-backtracking algorithm, bounded
  by `O(len(pattern) * len(path))`. Patterns are limited to 2 048 bytes and
  files to 10 000 rules, so a hostile robots.txt cannot make URL admission
  quadratic in the file size. A naive recursive matcher would take
  exponential time on patterns such as `*a*a*a...b$`;
  `TestPathologicalPatternIsFast` guards against that.
- Patterns and paths are compared after the same percent-encoding
  normalization, so `/caf%C3%A9` matches a link written as `/café`.
- Unknown directives (`Host`, `Noindex`, `Clean-param`) are reported but do
  not end a run of `User-agent` lines, matching Google's parser; otherwise a
  `Host:` line between two `User-agent` lines would split the group.

## Sitemaps and decompression limits

- Two layers of gzip exist and both are bounded. HTTP `Content-Encoding:
  gzip` is removed by the fetcher under its decompressed-size limit.
  Sitemap files that are themselves gzip archives (`sitemap.xml.gz` served
  as `application/gzip`) are decompressed by the sitemap package under the
  same 50 MiB limit. A gzip archive nested inside another is decompressed
  only once; the inner archive then fails XML parsing instead of being
  unpacked recursively.
- The sitemap limit reader reports overflow as an error instead of
  truncating like `io.LimitReader`, so a compression bomb is reported as a
  limit violation and not as "unexpected EOF". While writing the bomb test,
  the first version still classified any error on a `200` response as an
  invalid document (SEO-SITEMAP-003); only parse failures are now reported
  as invalid, and limit violations as unavailable (SEO-SITEMAP-002).
- XML is parsed with `encoding/xml` in strict mode. It does not resolve
  external entities or expand internal entity declarations, so XXE and
  "billion laughs" do not apply; a DOCTYPE with custom entities makes the
  document fail to parse, which is reported as an invalid sitemap.
- Sitemaps declared in robots.txt may live on other hosts (cross-submission)
  and are fetched through the same guarded dialer. URLs listed in them are
  only compared with the crawl when they are in scope.

## HTML parsing

- **golang.org/x/net/html** (already a dependency for IDNA) implements the
  HTML5 tree construction algorithm, so broken markup is interpreted as a
  browser would. `golang.org/x/net/html/charset` decodes legacy encodings
  (ISO-8859-2 is still common on Polish sites) from the Content-Type header,
  BOM or `<meta charset>`.
- **Parser failure:** `html.Parse` returns
  `open stack of elements exceeds 512 nodes` for documents nested deeper
  than 512 open elements. Browsers render such pages, but CrawlGrade cannot
  build a reliable model; the page is reported as unparseable instead of
  being analysed from a partial tree.
- Tree walks are iterative (explicit stack) so traversal cost does not
  depend on goroutine stack size.
- `<title>` and `<a>` inside SVG or MathML are foreign elements and are
  ignored; `<template>` contents are inert and skipped. A `<title>` outside
  `<head>` is used only when the head has none, which matches how browsers
  pick the document title.
- `<base href>` is honoured only for http(s) values; a `javascript:` base
  would otherwise make every relative link unresolvable.

## Robots metadata

- `<meta name="robots">`, `<meta name="googlebot">` and unprefixed or
  `googlebot:`-prefixed `X-Robots-Tag` headers are merged; the most
  restrictive value wins (`noindex` beats `index`, the smallest
  `max-snippet` wins, `-1` means unlimited). Directives for other named
  crawlers are recorded in the report but do not change indexability.
- `X-Robots-Tag: max-snippet: 0` must not be read as a directive for a
  crawler named "max-snippet"; names of parameterised directives are
  excluded from the agent-prefix detection.
- Robots meta tags in `<body>` still apply: search engines honour them, so
  ignoring them would hide a real `noindex`.
- **False positive decision:** `noindex` is reported as `low`, not higher,
  because excluding pages (search results, carts, thank-you pages) is
  routine. It is `critical` only on the audited start URL, and sitemap
  comparisons raise noindex URLs that are listed in a sitemap separately.

## Structured data

- JSON-LD is decoded with `encoding/json` (`UseNumber`, so large numbers are
  not rounded) after checking the block size (1 MiB). Trailing data after
  the top-level value is an error; browsers ignore it, but search engines
  reject the block.
- Type names are normalized from `Product`, `schema:Product` and
  `https://schema.org/Product`. Common subtypes (`BlogPosting`, `Optician`,
  `Store`, ...) map to the recognized type whose checks apply.
- **False positive decision:** recommended properties are only checked on
  top-level items. Nested entities such as an Article's `author` Person or a
  `publisher` Organization rarely carry logos and URLs, and flagging them
  would add a finding to nearly every article. Required properties are
  still checked on nested items.
- The walker visits object keys in sorted order and stops after 10 000 nodes
  or 32 levels, so evidence paths are deterministic and hostile blocks
  cannot make traversal unbounded.
- Script content is raw text in HTML: `&amp;` stays literal and a JSON
  string containing `</script>` must be escaped as `<\/script>` by the
  site. CrawlGrade does not HTML-decode JSON-LD.

## Hreflang, social metadata and links

- **hreflang.** A `<link rel="alternate" hreflang="…">` is recorded only when
  both `rel` contains `alternate` and an `hreflang` attribute is present, so a
  plain `rel="alternate"` to a print or AMP version is not mistaken for a
  language alternate. The value is validated as an ISO 639-1 code, optionally
  followed by a hyphen and an ISO 3166-1 region code, with `x-default` as the
  special fallback. A page that declares alternates without a self-reference
  is flagged, because search engines use the self-reference to confirm the
  page is one of the alternates.
- **Social preview.** Only `og:*` property tags and `twitter:*` meta tags are
  collected. Conflicting `og:title`/`og:description` and their Twitter
  equivalents are reported as `info` because most networks read the Open
  Graph tags. An image tag whose value is not an absolute http(s) URL is
  flagged; the pixel-size check is done later, in the link-graph analysis,
  where the corresponding `<img>` width and height are available.
- **Links.** Each `<a>` is resolved against the page base and classified as
  internal or external; `<img>` src values are recorded too, so broken and
  resource images are reported alongside broken links. Anchor text is the
  collapsed visible text, falling back to the alt text of a single contained
  image, matching the heading helper.
- **Site-wide graph.** The `links` package builds the directed link graph
  across all crawled pages. It reports indexable pages that no other page
  links to (orphans), self links, internal targets that were linked to but
  never fetched, and anchors reused for different targets across the site.
  Pages that are `noindex` are excluded from the orphan and repeated-anchor
  checks, since they carry no link authority to distribute. The analysis is a
  pure function of the visited pages, so it is deterministic and needs no
  network.
- **False positive decision:** orphans are reported only for indexable
  pages. A `noindex` page (a search-results page, a cart, a thank-you page)
  is routinely unlinked, and flagging it would add noise to every audit.

## Content, term strength and duplicates

- **Content extraction.** The `content` package parses the document and walks
  it with a bounded, stack-based traversal (so a hostile document cannot
  exhaust the stack). A fixed set of elements (`head`, `nav`, `header`,
  `footer`, `aside`, `form`, `script`, `style`, `noscript`, `template`, `svg`,
  `math`) is treated as shared chrome and skipped; the text of everything
  else is collected as main content. The collected text is capped at 64 KiB
  and the traversal at 20 000 elements. A page whose main text is less than
  5% of the raw text is reported as boilerplate; a page with no tokens left
  is reported separately.
- **Tokenization.** The `terms` package keeps runs of letters and digits and
  drops everything else. A small union of Polish and English stopwords
  removes the function words that carry no topical signal, so the same list
  works for mixed-language pages. Tokens longer than 40 runes are dropped so
  a hostile document cannot produce gigantic n-grams.
- **Term strength.** The `terms` package extracts unigrams through trigrams
  and, per page, ranks them by raw frequency. The `terms_strength` package
  aggregates the profiles across the crawl: a term's site-wide strength is the
  fraction of indexable pages that mention it. A term concentrated on one page
  is page-specific; a term spread across many pages is site-wide. Pages that
  share the same dominant n-grams are reported as reinforced duplicates.
- **Duplicates.** The `duplicates` package finds exact duplicates by hashing
  the extracted text and near-duplicates with a SimHash: a 64-bit fingerprint
  of the token stream that stays small when two documents share many tokens.
  Two pages are near-duplicates when their fingerprints are within a Hamming
  distance of 5, which tolerates the word reordering and small edits that
  separate real duplicates while keeping unrelated pages far apart. Both
  analyses are pure functions of the extracted text, so they are
  deterministic and need no network.
- **False positive decision:** duplicate and term-strength findings are
  reported only for indexable pages; a `noindex` page is not treated as a
  competitor for a topic.

## Reporting, audit orchestration and web hygiene

- **Report model.** `internal/report` defines a versioned, self-describing
  `Report` (`schema_version: "1"`) that is the single input to every renderer
  and to the baseline diff. Findings are grouped by the analysis that produced
  them (`crawlability`, `metadata`, `content`, `structured-data`,
  `internal-linking`, `web-hygiene`), and the group names are a public
  contract. `New` sorts the findings so every format and every diff is
  deterministic.
- **Renderers.** The terminal renderer emits severity markers and never emits
  ANSI colour itself (the caller may add it). The JSON renderer is indented
  and round-trips through `Load`. The HTML renderer is a single standalone
  document with inline CSS (light/dark mode) and no external resources; every
  value is passed through `html/template`, so a hostile title cannot inject
  markup.
- **Baseline and diff.** `Compare` matches findings by a key of
  `ID\x00URL\x00evidence` so the same observation is stable across runs. It
  errors when the baseline and current reports have different schema versions,
  and the CLI exposes the result through `--baseline` (full report) and
  `--diff` (only the delta).
- **Audit orchestration.** `internal/audit` is the only place that turns a URL
  into a report. It fetches robots.txt and a sitemap before the crawl so their
  rules govern the crawl, parses each HTML page with `htmlcheck` in the
  crawler's `Process` callback (storing the page and its scope on
  `crawler.Page.Data`), and then runs the per-page and site-wide analyses. The
  audit defaults the crawler's `MaxPages`, `MaxDepth` and `Concurrency` to the
  crawler's own defaults when the caller passes the zero value, because the
  crawler treats `MaxDepth: 0` as "fetch only the start URL".
- **Web hygiene.** `internal/webhygiene` runs passive checks against the
  headers and markup the audit already fetched, plus one extra request to the
  `http://` variant of the start URL to test the redirect to HTTPS. The
  `X-Content-Type-Options` check fires when the header is missing or not
  `nosniff`, matching the `Referrer-Policy` check, which fires when the header
  is absent. The three rules (`WEB-HYGIENE-001..003`) are reported separately
  from the SEO score.
- **False positive decision:** web hygiene describes properties visible to any
  visitor; it is not a vulnerability scan, and a clean result does not imply
  the site is secure.
- **SSRF interaction.** The audit runs the crawler with the guarded dialer and
  the default crawl-trap limits. Auditing a loopback or private target
  requires `--allow-private` (an environment variable in the first version,
  see the review below); otherwise the network policy blocks the requests and
  the report records the blocked robots.txt and sitemap.

## Review of milestones 3-5 (2026-09-15)

Before starting milestone 6, the code merged for milestones 3-5 was compared
with the specification and with this journal. Several statements above did
not match the code and are corrected here:

- The crawler clears `Response.Body` after `Process` returns. The audit then
  re-read the bodies for content, term and web hygiene analysis, so in real
  crawls those analyses received empty input. Unit tests passed because they
  exercised the packages directly.
- Structured data and breadcrumb findings were not wired into the audit
  (`structuredFindings` returned nil), and duplicate detection was not called.
- Web hygiene did not make the described extra request to the `http://`
  variant of the start URL; only HTTPS, `X-Content-Type-Options` and
  `Referrer-Policy` were checked.
- Term strength ranked raw frequencies. There were no zone weights, no
  document-frequency weighting and no 0-100 scale; the Polish stopword list
  was incomplete and contained non-words.
- `--allow-private` was an environment variable (`GR_ALLOW_PRIVATE`) rather
  than the flag the safety model documents.

`DEVELOPMENT_PLAN.md` now shows these items as open. They are completed in
the following commits before milestone 6 work starts.

Further problems found while fixing the items above:

- **robots.txt unavailable was treated as "allow everything".** The audit
  passed no `Allowed` callback when the robots.txt outcome was not `ok`, so a
  `503` robots.txt let the crawler fetch every page. RFC 9309 requires the
  opposite. The audit now always uses `robots.File.Allowed`, which already
  encoded the RFC policy.
- **The start URL bypassed robots.txt.** The crawler checked `Allowed` only
  for discovered links, never for the start URL itself. It now checks the
  start URL first and stops with `start-disallowed` without fetching it.


## Completion work (2026-09-22)

The open milestone review was used as the implementation checklist. Existing
uncommitted hreflang work was completed. The original SEO-HREFLANG-001 meaning
was retained; missing return links use the new SEO-HREFLANG-007 instead.

- Analyses now run inside the crawler callback while the decoded HTML tree
  exists. Content and structured-data findings survive body disposal. HTTP
  Link/X-Robots-Tag headers are applied before indexability checks, and
  relative references use the final URL. Duplicate analysis is connected.
- Sitemap directives feed discovery; sitemap-only URLs enter the bounded
  frontier after the start page. Origin-specific robots policies are cached
  for admitted origins. Redirect hops still use fetcher policy, without
  independent robots discovery. The whole audit, including discovery and
  optional external checks, now shares a maximum-duration context.
- Explicit depth zero stays zero. CLI values are validated before networking;
  blocked and failed starts produce documented exit codes. Flags cover output,
  JSON, keywords, timeout, user agent, rate, external HEAD checks and private
  networks. HTTP preview checks are bounded by the same fetcher and limiter.
- Unvisited links are no longer called broken. Self-links do not count as
  inbound links; the start URL is exempt from orphan findings. `noopener`
  and `noreferrer` no longer incorrectly suppress graph edges as nofollow.
- Main/article preference, hidden-node exclusion, corrected Polish stopwords,
  Unicode normalization and weighted TF-IDF scores are implemented. The
  formula, normalization and limitations are in `docs/analysis.md`.
- `golang.org/x/text` is now a direct dependency for language/region validation
  and NFC normalization. No new module was added.
- Exact duplicate grouping uses SHA-256. Near duplicates are grouped around
  representatives; this replaces repeated allocation of every matching pair.
- JSON-LD collection stops at the block count and byte limits before copying
  more scripts. Nested array traversal now obeys the depth guard too.
- Reports sort all finding groups, expose IDs, page/link metrics and terms,
  separate hygiene from SEO scoring, strip terminal controls, and apply a
  script-free CSP to standalone HTML. Baseline reports validate schema 1;
  offline diff includes aggregate deltas and full reports retain the delta.
- Tests caught the changed depth-zero semantics and confirmed actual content
  extraction after body disposal. Compiled CLI tests compare deterministic
  output and a normalized local-site golden; no test requires public Internet.
- Local `gofmt -l .` and `gosec ./...` also inspect ignored `.tools` and
  `.gopath` fixtures in this checkout. Project formatting is checked with
  `gofmt -l cmd internal test`; gosec excludes only those two cache directories.
  The local testsite intentionally serves hostile HTML; its G705 suppression
  documents that purpose. Baseline file reading is explicitly user-directed.
- HTML report sections (terms, quick wins, findings by severity, pages) are
  collapsible with standard `<details>` elements, summary headings and badge
  counts. Headline stats link directly to corresponding finding sections with
  smooth scrolling and `:target` highlight rings. A floating back-to-top button
  provides quick navigation back to the report header in light and dark modes.

