# Crawling, scope and safety model

CrawlGrade fetches pages from a site the user names on the command line. The
site's content is **untrusted input**: HTML, headers, redirects, robots.txt
and sitemaps may be written by an attacker who wants the crawler to hang,
exhaust memory, or reach a network the user did not intend to expose.

This document defines what the crawler may fetch and which limits apply.

## Goals

1. Audit a public website the way a well-behaved crawler sees it.
2. Never become a proxy into the user's private network (SSRF).
3. Keep memory, time and request volume bounded regardless of site content.
4. Be polite: respect robots.txt, identify itself, rate-limit requests.

## Non-goals

- Executing JavaScript or rendering pages.
- Authenticated crawling, form submission or any non-GET/HEAD request.
- Active security testing (no XSS, SQL injection, fuzzing or exploit
  payloads are ever sent). Web hygiene checks are passive and only read
  response headers and markup of pages that were fetched anyway.

## Privacy

- CrawlGrade sends no telemetry and contacts no host other than the target
  site (and, with `--check-external-links`, the external URLs that the site
  itself links to).
- Requests carry no cookies; cookies set by the site are discarded.
- Environment proxy settings (`HTTP_PROXY`, `HTTPS_PROXY`) are ignored,
  because a proxy would resolve names and connect on the crawler's behalf and
  defeat the destination checks below.
- Reports contain data from the audited site only.

## Scope

A URL is **in scope** when:

- its scheme is `http` or `https`, and
- its host equals the start URL's host, ignoring a leading `www.` on either
  side (`example.com` and `www.example.com` are the same site), and
- it passes robots.txt for that host.

Everything else is **external**. External URLs are recorded as link targets
but never crawled. With `--check-external-links` their status is checked with
a bounded number of HEAD (falling back to GET) requests, subject to the same
network policy.

## Network policy (SSRF protection)

Every connection, including each redirect hop, robots.txt, sitemaps, image
and external link checks, goes through a single guarded dialer:

1. The host name is resolved once by CrawlGrade.
2. **Every** returned address is classified. If any address is disallowed
   the host is rejected. Accepting only the "good" addresses of a mixed answer
   would let an attacker win by answering with one public and one private
   record.
3. The connection is made to the checked IP literal, not to the name, so a
   second resolution with a different answer cannot be used between the check
   and the connection (DNS rebinding).
4. The dialer's socket control hook checks the address actually being
   connected to once more, as defence in depth.

Redirects are followed manually, one hop at a time, so each hop's target is
subject to the same policy.

Disallowed by default:

| Range | Reason |
|-------|--------|
| `127.0.0.0/8`, `::1/128`, names `localhost` and `*.localhost` | loopback |
| `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7` | private networks |
| `169.254.0.0/16`, `fe80::/10` | link-local (includes cloud metadata) |
| `100.64.0.0/10` | carrier-grade NAT (includes some metadata services) |
| `0.0.0.0/8`, `::/128`, multicast, broadcast, `240.0.0.0/4` | unspecified or reserved |
| documentation, benchmarking and IETF protocol ranges | reserved |
| IPv4-mapped, NAT64, 6to4 addresses embedding any of the above | tunnelled private addresses |

`--allow-private` permits loopback, private, CGNAT and link-local unicast
addresses so that local development sites can be audited. Cloud metadata
endpoints (`169.254.169.254`, `169.254.170.2`, `fd00:ec2::254`,
`100.100.100.200`) and unspecified/multicast addresses remain blocked even
then.

## Limits

All limits are enforced by the crawler; flags adjust the defaults.

| Limit | Default | Flag |
|-------|---------|------|
| Pages fetched | 500 | `--max-pages` |
| Link depth from the start URL | 10 | `--max-depth` |
| Concurrent requests | 4 | `--concurrency` |
| Requests per second (global) | 5 | `--requests-per-second` |
| Per-request timeout | 15s | `--timeout` |
| Whole crawl duration | 10m | `--max-duration` |
| Redirect hops per request | 10 | |
| Response body (on the wire) | 5 MiB | |
| Decompressed body | 10 MiB | |
| robots.txt | 500 KiB | |
| Sitemap file (decompressed) | 50 MiB | |
| Sitemap files | 50 | |
| URLs taken from sitemaps | 50 000 | |
| Frontier queue | 10 x max pages (1 000 - 100 000) | |
| URL length | 2 048 bytes | |

Hitting a limit never crashes the crawl: the affected response or URL is
recorded with the reason and the crawl continues or stops cleanly with a
partial report.

## Crawl traps

- **Depth and page limits** bound every trap eventually.
- **Query explosions:** at most 25 distinct query strings are crawled per
  path; further variants are skipped and reported.
- **Infinite calendars and generated paths:** paths are reduced to a pattern
  (digit runs replaced by a placeholder). At most 100 URLs per pattern are
  crawled.
- **Repeating path segments** (`/a/b/a/b/a/b/`): a segment occurring more
  than 3 times in a path marks the URL as a trap.
- **Redirect loops** are detected per request and reported; a loop never
  consumes more than the redirect hop limit.

## Robots

robots.txt is fetched for each in-scope host before any page on it, following
RFC 9309:

- `4xx` (including 404): no restrictions. A missing robots.txt is not an
  error and is reported only as information.
- `5xx`, network failure, or a redirect loop: the host is treated as fully
  disallowed, as RFC 9309 requires.
- Groups for the `crawlgrade` product token take precedence over `*`.
- Longest matching path wins; on equal length `Allow` wins. `*` and `$`
  wildcards are supported.

## Cancellation

The crawl runs under a context. Ctrl-C, `--max-duration` or an internal error
cancel in-flight requests, stop scheduling new ones, and produce a partial
report that states why the crawl stopped.
