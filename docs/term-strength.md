# Term Strength and Keyword Analysis

CrawlGrade analyzes textual relevance and topic prominence across the audited
site using a transparent, deterministic TF-IDF model.

It measures which terms and multi-word phrases (n-grams) dominate individual pages
and the site as a whole. It is **never** referred to as a "Google keyword score"
and does not predict search engine rankings.

---

## 1. Text Extraction and Tokenization

Before measuring term strength, HTML markup is stripped down to meaningful
content zones:
- Scripts, styles, `<noscript>`, SVG, forms, and template tags are discarded.
- Main content extraction prioritizes semantic `<main>` or `<article>` tags,
  dropping site chrome (navbars, headers, footers, sidebars).
- Text is normalized to Unicode NFC and lowercased.
- Characters outside Unicode letters and digits are treated as token boundaries.
- Individual tokens longer than 40 runes are discarded to guard against hostile
  content explosions.

### Bilingual Stopword Filtering

CrawlGrade embeds static stopword sets for **Polish** and **English** in
`internal/terms/token.go`. Function words (articles, prepositions, auxiliary
verbs, pronouns, common conjunctions) are removed before scoring.

Because CrawlGrade does not perform runtime network lookups or heavy NLP
models, stopword removal is purely deterministic and fast. When a stopword is
dropped between two topical words, they can join to form informative n-grams.

---

## 2. N-Gram Generation

CrawlGrade extracts:
- **Unigrams**: single words (e.g. `optometrysta`, `audit`)
- **Bigrams**: two consecutive tokens (e.g. `badanie wzroku`, `crawl analysis`)
- **Trigrams**: three consecutive tokens (e.g. `okulary korekcyjne progresywne`, `technical seo audit`)

N-grams are bounded between 1 and 3 tokens.

---

## 3. Zone Weighting

Not all text on a page has equal topical significance. CrawlGrade applies zone
multipliers when calculating term frequencies:

| Content Zone | Weight Multiplier | Rationale |
|---|---|---|
| `<title>` tag | **4×** | Primary document topic indicator |
| Headings (`<h1>`–`<h6>`) | **3×** | Sectional hierarchy and key topics |
| Meta Description | **2×** | Summary snippet intended for search results |
| Main Body Content | **1×** | Baseline article and paragraph text |

---

## 4. Transparent TF-IDF Formulation

For each n-gram term $t$:
- $\text{TF}(t)$: Total weighted occurrences of $t$ across the zones of a page.
- $\text{DF}(t)$: Number of indexable crawled pages on which $t$ appears.
- $N$: Total number of indexable pages analyzed in the crawl.

### Mathematical Formulation:

$$\text{Logarithmic TF}(t) = 1 + \ln(\text{TF}(t))$$

$$\text{Smoothed IDF}(t) = 1 + \ln\left(\frac{N + 1}{\text{DF}(t) + 1}\right)$$

$$\text{Raw Weight}(t) = (1 + \ln(\text{TF}(t))) \times \left(1 + \ln\left(\frac{N + 1}{\text{DF}(t) + 1}\right)\right)$$

### Role of Document Frequency ($\text{DF}$):
Document frequency prevents repetitive website boilerplate (e.g., brand names,
standard copyright notices, or navigation menu links that escaped extraction)
from dominating topical scores. Terms present on almost every page receive a low
IDF multiplier, whereas terms concentrated on specific pages score high.

---

## 5. Normalization (0–100)

Raw weights vary based on document length and term counts. To make results
intuitive and comparable:
- On each page, the highest scoring term is normalized to **100.0**.
- All other terms on that page are scaled proportionally:
  $$\text{Normalized Score}(t) = \text{round}\left(100.0 \times \frac{\text{Raw Weight}(t)}{\max_i \text{Raw Weight}(i)}\right)$$
- Site-wide scores average the raw weights across all crawled pages, then normalize
  the top site term to **100.0**.

---

## 6. User-Specified Keywords

Users can supply target keywords via CLI flags:

```sh
crawlgrade https://example.com --keywords 50
```

Requested keywords are tokenized with the same rules and tracked alongside the
top 20 discovered terms. If a requested keyword appears on a page, its relative
strength is displayed; if absent, it receives a score of 0.0.
