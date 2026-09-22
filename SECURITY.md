# Security

CrawlGrade performs passive technical audits. It does not send attack payloads,
execute JavaScript, authenticate to sites, submit forms or predict rankings.
The security boundary and network restrictions are documented in
[docs/crawling.md](docs/crawling.md).

For a suspected SSRF bypass, parser crash or unsafe report rendering, use the
repository's private vulnerability reporting facility. Include the commit or
version, a minimal local fixture, reproduction commands and observed behavior.
Avoid disclosing credentials or private site data in public issues.

Development follows the current main branch. Release support and published
artifacts are established through the release workflow. There is no claim of
an independent penetration test or security certification.
