# Limitations and Boundaries

CrawlGrade is designed with explicit boundaries to ensure safety, determinism,
and predictable resource usage. This document details intentional design
trade-offs and non-goals.

---

## 1. No Active Security Testing

CrawlGrade is **not** a penetration testing tool or vulnerability scanner:
- It never sends exploit payloads (such as SQL injection, XSS probes, or path traversal).
- It never fuzzes endpoints, brute-forces directories, or fuzzes parameters.
- It never submits forms or modifies state on the target server.
- The passive web hygiene checks evaluate only standard HTTP response headers and
  markup visible to any regular visitor. A clean web hygiene report does **not**
  mean a site is secure.

---

## 2. No JavaScript Execution (Static Parsing Only)

CrawlGrade parses raw HTTP response bodies using an HTML token stream parser:
- It does **not** run a headless browser (such as Chromium, Playwright, or Puppeteer).
- It does **not** execute client-side JavaScript or DOM manipulations.
- Single Page Applications (SPAs) that require client-side JavaScript to render
  content or inject navigation links will appear largely empty to CrawlGrade unless
  pre-rendered via Server-Side Rendering (SSR) or dynamic rendering.

---

## 3. No Ranking Prediction or SERP Simulation

CrawlGrade audits technical SEO foundations, not search engine algorithms:
- It never predicts search engine rankings, Google SERP positions, or click-through rates.
- Term strength scores describe textual prominence within the audited site's corpus;
  they are not competitive keyword rankings or search volume estimates.
- Search engines incorporate hundreds of external factors (search intent, user behavior,
  backlink authority, localization) that a passive site crawler cannot measure.

---

## 4. Crawl Bounds and Safety Limits

To protect both the crawler host and the audited server, CrawlGrade enforces
strict bounds:

| Boundary | Default Limit | Purpose |
|---|---|---|
| Maximum Pages | 100 pages (`--max-pages`) | Prevents unbounded crawling on large sites |
| Maximum Depth | 5 hops (`--max-depth`) | Prevents deep rabbit-hole link exploration |
| Request Timeout | 10 seconds (`--timeout`) | Prevents stalled connections |
| Max Response Size | 2 MiB per response | Blocks memory exhaustion from giant HTML |
| Max Decompression | 8 MiB per response | Defends against gzip bombs |
| Concurrency | Bounded worker pool | Controls memory and CPU consumption |
| Rate Limiter | Configurable RPS | Prevents accidental denial-of-service |
| Crawl Traps | Max 3 repeating segments | Defends against infinite calendars and query explosions |

---

## 5. SSRF Defense and Private Networks

CrawlGrade enforces strict SSRF protections via `netguard`:
- Localhost (`127.0.0.0/8`, `::1`), RFC 1918 private subnets, link-local addresses
  (`169.254.0.0/16`), and cloud metadata IP endpoints (`169.254.169.254`) are blocked
  by default.
- Every redirect hop re-resolves and validates the destination IP before dialing,
  mitigating DNS rebinding attacks.
- The `--allow-private` flag permits local testing against loopback/RFC 1918 ranges,
  but cloud metadata endpoints remain strictly blocked at all times.

---

## 6. Duplicate Content Heuristics

Duplicate content detection uses deterministic heuristics:
- **Exact Duplicates**: Identified using SHA-256 digests of extracted main text.
- **Near Duplicates**: Identified using 64-bit SimHash fingerprints with a Hamming
  distance threshold of 5.
- Output uses representative grouping ($O(N)$ representation) rather than outputting
  all $O(N^2)$ pairwise combinations.
- Minor differences in whitespace or excluded navigation do not influence the hash.
