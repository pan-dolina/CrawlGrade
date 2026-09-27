# Rule Catalog and Checks

CrawlGrade organizes its findings into six public categories. Every finding
carries an immutable ID, a severity, an affected URL, category, title, evidence,
and remediation guidance.

Finding IDs follow a strict format (`SEO-<GROUP>-<NUMBER>` or `WEB-HYGIENE-<NUMBER>`)
and are a public contract. They are never renamed, renumbered, or reused.

---

## Severity Scale

Findings are categorized into five severity levels:

| Severity | Description | Penalty in SEO Score |
|---|---|---|
| **Critical** | Major issue preventing search engines from accessing or indexing key content | -20 pts / page |
| **High** | Substantial technical defect impairing discovery, crawling, or indexation | -8 pts / page |
| **Medium** | Notable technical deficiency that dilutes signal or violates standards | -3 pts / page |
| **Low** | Minor omission, suboptimal attribute, or technical hygiene gap | -1 pt / page |
| **Info** | Informational observation, heuristic signal, or best practice recommendation | 0 pts |

---

## 1. Crawlability

Checks addressing how crawlers discover, retrieve, and traverse pages across the site.

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `SEO-CRAWL-001` | Info | Depth limit reached | Crawl reached maximum configured depth (`--max-depth`). |
| `SEO-CRAWL-002` | High | Crawl trap pattern detected | URL contains repeating path segments or recursive calendar structures. |
| `SEO-CRAWL-003` | High | Redirect loop detected | URL redirects in a circular sequence back to an earlier URL. |
| `SEO-CRAWL-004` | Medium | Redirect chain detected | URL took more than 2 redirect hops to reach its destination. |
| `SEO-CRAWL-005` | High | Decompression limit exceeded | Response decompressed size exceeded safety limit (gzip bomb guard). |
| `SEO-CRAWL-006` | High | Response size limit exceeded | Response body size exceeded maximum configured byte limit. |
| `SEO-HTTP-001` | High | HTTP 4xx/5xx status error | Server returned a client or server error status code. |
| `SEO-HTTP-002` | Low | Unsupported or missing Content-Type | Response lacks a declared MIME type or is not parseable HTML. |
| `SEO-ROBOTS-001` | Medium | robots.txt unreachable / server error | Fetching `robots.txt` returned a 5xx error or connection failure. |
| `SEO-ROBOTS-002` | Low | robots.txt syntax error | Malformed lines or unparseable directives in `robots.txt`. |
| `SEO-ROBOTS-003` | Info | Page disallowed by robots.txt | Page matched a Disallow rule for user-agent. |
| `SEO-SITEMAP-001` | Low | Sitemap not declared in robots.txt | `robots.txt` does not reference any Sitemap directive. |
| `SEO-SITEMAP-002` | Medium | Sitemap fetch error | Sitemap URL failed to fetch or returned an error status. |
| `SEO-SITEMAP-003` | Medium | Sitemap XML parse error | Sitemap content is malformed XML or invalid gzip. |
| `SEO-SITEMAP-010` | High | Sitemap URL error or mismatch | A URL listed in the sitemap returned 4xx/5xx or redirects elsewhere. |

---

## 2. Metadata

Checks analyzing `<title>`, `<meta>`, canonical links, headings, language, and social cards.

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `SEO-TITLE-001` | High | Page is missing a title tag | HTML document contains no `<title>` element. |
| `SEO-TITLE-002` | High | Page has an empty title tag | `<title>` element exists but contains only whitespace. |
| `SEO-TITLE-003` | Medium | Page has multiple title tags | HTML document contains more than one `<title>` element. |
| `SEO-TITLE-004` | Medium | Duplicate title tag across pages | Multiple crawled pages share the exact same title text. |
| `SEO-TITLE-005` | Low | Title tag is unusually short | Title text has fewer than 10 characters. |
| `SEO-TITLE-006` | Low | Title tag is unusually long | Title text exceeds 70 characters. |
| `SEO-DESC-001` | Medium | Page is missing a meta description | No `<meta name="description">` tag present. |
| `SEO-DESC-002` | Medium | Page has an empty meta description | Meta description tag has empty or whitespace-only content. |
| `SEO-DESC-003` | Low | Multiple meta descriptions declared | Document contains more than one meta description tag. |
| `SEO-DESC-004` | Low | Duplicate meta description across pages | Multiple crawled pages share the exact same description. |
| `SEO-DESC-005` | Low | Meta description length suboptimal | Description length is outside recommended bounds (< 50 or > 160 chars). |
| `SEO-CANONICAL-001` | Medium | Missing canonical link tag | Page has no `<link rel="canonical">` or canonical Link header. |
| `SEO-CANONICAL-002` | High | Multiple canonical URLs declared | Multiple conflicting canonical links found in HTML or headers. |
| `SEO-CANONICAL-003` | Low | Relative canonical URL declared | Canonical href is relative instead of absolute URL. |
| `SEO-CANONICAL-004` | High | Canonical target returns error | Canonical URL points to a broken page (4xx/5xx). |
| `SEO-CANONICAL-005` | High | Canonical loop detected | Canonical links form a circular reference. |
| `SEO-CANONICAL-006` | Medium | Canonical chain detected | Canonical target specifies a different canonical URL. |
| `SEO-CANONICAL-007` | Info | Cross-domain canonical declared | Canonical link points to a different origin/domain. |
| `SEO-CANONICAL-008` | Medium | Canonical target redirects | Canonical URL responds with a 3xx redirect. |
| `SEO-CANONICAL-009` | Low | Canonical declared outside `<head>` | Canonical link is located in `<body>`, which crawlers may ignore. |
| `SEO-INDEX-001` | Info | Page marked noindex | Page sends `noindex` via meta tag or `X-Robots-Tag`. |
| `SEO-INDEX-002` | Low | Page marked nofollow | Page sends `nofollow` via meta tag or `X-Robots-Tag`. |
| `SEO-INDEX-003` | High | Conflicting robots directives | Document specifies conflicting directives (e.g. `index` and `noindex`). |
| `SEO-INDEX-004` | Low | Invalid robots directive | Document specifies unparseable robots directive tokens. |
| `SEO-INDEX-005` | Info | Snippet/preview restrictions declared | Directives like `nosnippet`, `max-snippet`, or `max-image-preview` used. |
| `SEO-INDEX-006` | High | Start URL is marked noindex | Audit root / start URL is blocked from indexation with `noindex`. |
| `SEO-INDEX-007` | High | Canonical target is marked noindex | Canonical URL points to a target that is disallowed from indexation. |
| `SEO-HEADING-001` | Medium | Missing `<h1>` heading | Document contains no `<h1>` heading tag. |
| `SEO-HEADING-002` | Low | Empty heading element | Heading tag (`<h1>`-`<h6>`) contains no text. |
| `SEO-HEADING-003` | Low | Multiple `<h1>` headings present | Document contains more than one `<h1>` element. |
| `SEO-HEADING-004` | Low | Duplicate `<h1>` across pages | Multiple crawled pages share the exact same `<h1>` text. |
| `SEO-HEADING-005` | Low | Broken heading hierarchy | Headings skip levels (e.g., `<h1>` followed by `<h3>` without `<h2>`). |
| `SEO-LANG-001` | Low | Missing html lang attribute | The root `<html>` element does not specify a `lang` attribute. |
| `SEO-LANG-002` | Low | Invalid html lang value | `lang` attribute contains an invalid BCP 47 language code. |
| `SEO-LANG-003` | Low | Language code mismatch with hreflang | Page `lang` does not match its corresponding hreflang self-reference. |
| `SEO-HREFLANG-001` | Medium | Missing hreflang self-reference | Page declares alternate languages but lacks a return link to itself. |
| `SEO-HREFLANG-002` | Low | Duplicate hreflang declaration | Same language code declared more than once on the page. |
| `SEO-HREFLANG-003` | Low | Invalid hreflang code | `hreflang` attribute has an invalid language or region tag. |
| `SEO-HREFLANG-004` | Info | Missing x-default fallback | Multilingual set does not designate an `x-default` fallback target. |
| `SEO-HREFLANG-005` | Medium | Missing reciprocal return hreflang | Target alternate page does not link back to the referencing page. |
| `SEO-SOCIAL-001` | Info | Missing OpenGraph metadata | Document lacks basic `og:title`, `og:description`, or `og:url` tags. |
| `SEO-SOCIAL-002` | Low | Missing OpenGraph image | Document defines OpenGraph metadata but lacks `og:image`. |
| `SEO-SOCIAL-003` | Info | Missing Twitter card metadata | Document lacks Twitter/X card tags (`twitter:card`). |
| `SEO-SOCIAL-004` | Low | Inconsistent social title/description | OpenGraph tags conflict substantially with HTML `<title>` or description. |
| `SEO-SOCIAL-005` | Low | Incomplete social card tags | Partial Twitter or OpenGraph card tags missing required properties. |
| `SEO-IMAGE-001` | Low | Image missing alt attribute | `<img>` tag lacks an `alt` attribute. |
| `SEO-IMAGE-002` | Info | Image missing dimensions | `<img>` tag lacks explicit `width` or `height` layout attributes. |
| `SEO-IMAGE-003` | Low | Invalid image loading attribute | `loading` attribute is neither `lazy` nor `eager`. |

---

## 3. Content

Checks evaluating body content extraction, page substance, boilerplate ratio, and duplication.

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `SEO-CONTENT-001` | Medium | Thin content / low text volume | Extracted main body text has fewer than 100 words. |
| `SEO-CONTENT-002` | Medium | High boilerplate-to-content ratio | Navigation, chrome, or repeated boilerplate dominates page text. |
| `SEO-TERMS-001` | Info | No strong terms identified | Content extraction yielded no prominent topical terms. |
| `SEO-TERMS-002` | Low | Strongest terms lack focus | Highest weighted terms are fragmented or generic. |
| `SEO-TERMS-003` | Low | Potential keyword competition | Multiple distinct pages compete for the exact same dominant terms. |
| `SEO-DUPLICATE-001` | High | Exact duplicate content | Two or more indexable pages have identical SHA-256 content hashes. |
| `SEO-DUPLICATE-002` | Medium | Near-duplicate content | Two or more indexable pages have SimHash Hamming distance <= 5. |

---

## 4. Structured Data

Checks evaluating JSON-LD schemas and BreadcrumbList markup.

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `SEO-SCHEMA-001` | High | JSON-LD syntax error | `<script type="application/ld+json">` contains invalid JSON syntax. |
| `SEO-SCHEMA-002` | Medium | Missing schema @context or @type | JSON-LD object lacks required `@context` or `@type` property. |
| `SEO-SCHEMA-003` | Low | Untyped schema object | Schema structure does not designate a known Schema.org type. |
| `SEO-SCHEMA-004` | Medium | Missing required schema property | Standard recognized schema type lacks mandatory core properties. |
| `SEO-SCHEMA-005` | Low | Missing recommended schema property | Standard schema type lacks recommended properties. |
| `SEO-SCHEMA-006` | Low | Invalid URL in schema | Schema property contains an unparseable or malformed URL. |
| `SEO-BREADCRUMB-001` | Low | Breadcrumb position discontinuity | `ListItem.position` numbers skip integers (e.g. 1 then 3). |
| `SEO-BREADCRUMB-002` | Low | Duplicate breadcrumb position | Multiple breadcrumb items claim the exact same position index. |
| `SEO-BREADCRUMB-003` | Low | Invalid breadcrumb item URL | Breadcrumb item lacks a valid target URL. |

---

## 5. Internal Linking

Checks analyzing internal link architecture, anchor text, and graph properties.

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `SEO-LINK-001` | High | Internal broken link | Internal link points to an in-scope URL returning 4xx or 5xx. |
| `SEO-LINK-002` | Medium | Potential orphan page | Crawled indexable page has 0 internal inbound links from other pages. |
| `SEO-LINK-003` | Info | Self-linking URL | Page contains a link pointing directly to its own canonical URL. |
| `SEO-LINK-004` | Low | Duplicate identical links on page | Page links to the exact same destination multiple times. |
| `SEO-LINK-005` | Low | External link missing noopener | External link opening in new window lacks `rel="noopener"`. |
| `SEO-LINK-008` | Low | External link check failed | Optional external link check returned an HTTP failure or unreachable host. |
| `SEO-ANCHOR-001` | Low | Empty anchor text | Link tag `<a>` has no visible text or descriptive image alt text. |
| `SEO-ANCHOR-002` | Low | Ambiguous anchor text | Identical anchor text links to different destinations across the site. |

---

## 6. Web Hygiene

Passive security and transport observations visible to any standard web visitor.
**Reported separately from the SEO score.**

| Rule ID | Severity | Title | Trigger Condition |
|---|---|---|---|
| `WEB-HYGIENE-001` | Medium | Page is served over plain HTTP | Final page URL uses `http://` instead of secure `https://`. |
| `WEB-HYGIENE-002` | Low | Missing X-Content-Type-Options: nosniff | Response does not declare `X-Content-Type-Options: nosniff`. |
| `WEB-HYGIENE-003` | Low | Missing Referrer-Policy header | Response lacks a declared `Referrer-Policy` header. |
| `WEB-HYGIENE-004` | Low | HTTP to HTTPS redirect not confirmed | HTTP variant did not redirect to HTTPS or failed to respond. |
| `WEB-HYGIENE-005` | Medium | Mixed content: HTTP resources referenced | HTTPS page references subresources (images, scripts) via plain `http://`. |
| `WEB-HYGIENE-006` | Low | Missing Strict-Transport-Security (HSTS) | HTTPS response lacks `Strict-Transport-Security` with `max-age`. |
| `WEB-HYGIENE-007` | Low | Missing Content-Security-Policy (CSP) | Response does not provide a `Content-Security-Policy` header. |
| `WEB-HYGIENE-008` | Low | Missing clickjacking protection | Response lacks `X-Frame-Options` and CSP `frame-ancestors`. |
