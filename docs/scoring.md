# Scoring Methodology

CrawlGrade provides two separate scores:
1. **Technical SEO Score** (0–100)
2. **Passive Web Hygiene Score** (0–100)

These scores summarize technical audit health and standards compliance. They
are descriptive heuristics, **not** search engine ranking predictions or security
certifications.

---

## Separation of Concerns

Web hygiene is strictly separated from the Technical SEO score:
- **Technical SEO Score**: Aggregates findings from Crawlability, Metadata, Content,
  Structured Data, and Internal Linking.
- **Web Hygiene Score**: Aggregates passive observations regarding HTTPS, secure
  transport (HSTS), MIME-type sniffing protection, referrer leakage, clickjacking
  defenses, and Content Security Policy.

A site with security hygiene issues does not artificially penalize its SEO content
score, and conversely, a clean security configuration does not mask missing titles,
broken links, or canonical loops.

---

## Penalty Weights

Each finding contributes a penalty based on its severity:

| Severity | Penalty Points | Description |
|---|---|---|
| **Critical** | 20 points | Severe failure (e.g. start URL noindexed, fatal crawler block) |
| **High** | 8 points | Serious defect (e.g. missing title, broken internal link, duplicate content) |
| **Medium** | 3 points | Notable issue (e.g. missing description, missing canonical, thin content) |
| **Low** | 1 point | Suboptimal configuration (e.g. empty anchor, title length heuristic) |
| **Info** | 0 points | Informational signal; does not affect the numerical score |

---

## Formula

The score normalizes accumulated penalties across the total number of indexable
pages ($N$) analyzed during the crawl:

$$\text{SEO Score} = \max\left(0, 100 - \left\lfloor \frac{\sum \text{SEO Penalties}}{N} \right\rfloor \right)$$

$$\text{Web Hygiene Score} = \max\left(0, 100 - \left\lfloor \frac{\sum \text{Hygiene Penalties}}{N} \right\rfloor \right)$$

### Properties and Edge Cases:
- **Indexable Pages Baseline**: $N$ is the number of crawled pages that are indexable
  (not marked `noindex` or blocked by robots).
- **Zero Indexable Pages**: If no indexable pages were parsed (e.g., the site blocked
  crawling or every page returned an error), no score is emitted (`null` / empty).
- **Floor Division**: Integer division is used, avoiding fractional point distractions.
- **Lower Bound**: The score is bounded at a minimum of `0`. It never produces negative
  scores.

---

## Interpretation Guidelines

- **90–100**: Strong technical foundation. Minimal or minor defects.
- **70–89**: Moderate technical health. Some high or medium findings require attention.
- **50–69**: Substantial technical friction. Multiple broken links, duplicate pages, or
  metadata omissions dilute crawling efficiency.
- **< 50**: Critical crawlability or structural breakdown. Immediate remediation
  recommended.

Always inspect the underlying finding list rather than relying exclusively on the
single summary number.
