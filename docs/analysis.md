# Analysis and interpretation

Content extraction prefers the first `main` or `article` region; otherwise it
uses the document. It excludes navigation, headers, footers, sidebars, forms,
scripts, styles, templates, foreign markup and hidden subtrees. It is a
documented heuristic, not a rendered-browser or machine-learning extractor.
Text is limited to 64 KiB and extraction visits at most 20,000 nodes before
skipping further subtrees. Pages consisting mainly of excluded text are
flagged as boilerplate. JavaScript is never executed.

Tokens are NFC-normalized Unicode letters and digits, lowercased, with a
static English/Polish stopword union in `internal/terms/token.go`. Tokens
longer than 40 runes are ignored. There is no stemming or language detection.
N-grams contain one through three remaining tokens; stopword removal can
therefore join words originally separated by a function word.

Term frequency uses weights: title 4, headings 3, description 2, main text 1.
For a term with weighted frequency TF and document frequency DF among N
indexable pages, its raw score is `(1 + ln(TF)) * (1 + ln((N+1)/(DF+1)))`.
Per-page scores normalize the strongest term to 100. Site scores average raw
weights across pages, then normalize to 100. Reports show 20 top terms plus
up to 100 requested keywords (zero when absent). These scores describe the
observed content, not ranking potential or cross-site competitiveness.

Exact duplicates use SHA-256 of extracted text. Near duplicates compare
64-bit SimHash against existing representatives with Hamming distance at most
5. Empty and noindex pages are excluded. Representative grouping keeps output
linear in the number of pages; it does not enumerate all similar pairs.

Internal inbound/outbound counts use distinct non-self, non-nofollow internal
targets. Inbound counts exclude noindex source pages. Orphan findings describe
only the observed graph; sitemap-only pages can be discovered, and the start
page is exempt. Unvisited targets are unknown, not broken. HTTP failures are
reported only after a request. `noopener` and `noreferrer` do not imply
`nofollow`. External checks are optional HEAD requests and may produce false
positives when a destination rejects HEAD.

Hreflang validates language, optional script and country codes, duplicates,
self-reference, x-default and return links for crawled targets. Missing
x-default is informational. No hreflang is required for a monolingual page.
The `html lang` attribute is validated independently. Images distinguish
missing alt from deliberately empty decorative alt; width, height and loading
attributes are checked without decoding image bytes. Internal image URLs are
admitted under the same crawl limits. Social checks validate basic declared
properties and image URLs; image pixel dimensions and rich-result eligibility
are not independently certified.

The technical SEO and passive hygiene scores are separate descriptive
summaries. Each is `max(0, 100 - floor(total penalties / parsed indexable pages))`:
critical 20, high 8, medium 3, low 1, info 0. No score is emitted when there
are no parsed indexable pages. Inspect individual findings and crawl coverage;
the arithmetic is deliberately simple and is not a search-engine model.
