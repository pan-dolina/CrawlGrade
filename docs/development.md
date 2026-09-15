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
