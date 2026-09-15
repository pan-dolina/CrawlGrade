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
