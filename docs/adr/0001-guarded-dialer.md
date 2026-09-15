# ADR 0001: Enforce the network policy in the dialer

## Status

Accepted

## Context

CrawlGrade follows URLs and redirects chosen by the audited site. Without
checks, `https://evil.example/` could redirect to
`http://169.254.169.254/latest/meta-data/` or `http://127.0.0.1:6379/`, or its
DNS name could resolve to a private address, turning the crawler into an SSRF
proxy into the user's network.

Checking URLs before a request is not enough:

- a host name can resolve differently at check time and at connection time
  (DNS rebinding);
- redirects are followed by `net/http` internally;
- environment proxies connect on the client's behalf.

## Decision

The policy is enforced where the connection is made. `netguard.Dialer`
resolves the name itself, rejects the host if any resolved address is
disallowed, and connects to the checked IP literal. A `net.Dialer.Control`
hook re-checks the socket address. The HTTP transport uses this dialer, has
no proxy, and `CheckRedirect` stops `net/http` from following redirects so
the fetcher follows (and records) every hop itself.

## Consequences

- One code path protects pages, redirects, robots.txt, sitemaps and link
  checks.
- Happy Eyeballs across multiple addresses is lost only to the extent that
  the dialer tries the checked addresses in order.
- TLS still verifies the certificate against the original host name because
  the transport's `TLSClientConfig.ServerName` is derived from the URL, not
  the dialed address.
