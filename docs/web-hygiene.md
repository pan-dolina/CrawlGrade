# Web Hygiene Checks

Web hygiene is reported **separately** from the SEO score. It describes passive,
security-relevant properties that are visible to any standard web visitor.

All checks are strictly passive: CrawlGrade inspects only the response headers
and HTML markup of pages already fetched during the audit, plus at most one
extra request to the `http://` variant of the start URL to confirm the HTTP to
HTTPS redirect. No attack payloads, fuzzing inputs, or invasive probes are ever
sent.

> **Disclaimer**: This is not a vulnerability assessment or penetration test. A
> clean web hygiene report does not indicate that a site is secure against web
> vulnerabilities.

---

## Catalog of Web Hygiene Checks

### 1. `WEB-HYGIENE-001`: Plain HTTP Transport
- **Severity**: Medium
- **Description**: The page was fetched over unencrypted plain `http://` instead
  of `https://`. In-transit traffic can be intercepted, inspected, or modified.
- **Remediation**: Serve the entire website over HTTPS with a valid TLS certificate.

### 2. `WEB-HYGIENE-002`: Missing `X-Content-Type-Options: nosniff`
- **Severity**: Low
- **Description**: The response lacks `X-Content-Type-Options: nosniff`. Browsers
  may attempt to MIME-sniff response bodies and execute them contrary to declared
  MIME types.
- **Remediation**: Configure the web server or application gateway to emit
  `X-Content-Type-Options: nosniff` on all responses.

### 3. `WEB-HYGIENE-003`: Missing `Referrer-Policy` Header
- **Severity**: Low
- **Description**: The response does not define a `Referrer-Policy`. Browsers may
  leak full URL paths or sensitive query parameters to third-party domains when
  users navigate away from the page.
- **Remediation**: Specify a secure policy such as
  `Referrer-Policy: strict-origin-when-cross-origin` or `no-referrer`.

### 4. `WEB-HYGIENE-004`: HTTP to HTTPS Redirect Not Confirmed
- **Severity**: Low
- **Description**: Connecting to the plain `http://` version of the start URL did
  not successfully redirect to its canonical HTTPS counterpart, or the response
  was unavailable.
- **Remediation**: Ensure port 80 / HTTP traffic sends a permanent redirect (`301`
  or `308`) directly to the HTTPS URL.

### 5. `WEB-HYGIENE-005`: Mixed Content (HTTP Resources on HTTPS Page)
- **Severity**: Medium
- **Description**: An HTTPS page references subresources (images, scripts, styles,
  iframes, or audio/video) via unencrypted `http://` URLs. Modern browsers block
  or upgrade mixed content.
- **Remediation**: Update resource references to use secure HTTPS URLs or root-relative
  paths.

### 6. `WEB-HYGIENE-006`: Missing `Strict-Transport-Security` (HSTS)
- **Severity**: Low
- **Description**: The HTTPS response does not send a `Strict-Transport-Security`
  header containing `max-age`. Browsers cannot enforce HTTPS on future visits,
  leaving initial requests vulnerable to SSL stripping attacks. (Per RFC 6797 §7.2,
  this check only applies to HTTPS pages).
- **Remediation**: Send `Strict-Transport-Security: max-age=31536000; includeSubDomains`.

### 7. `WEB-HYGIENE-007`: Missing `Content-Security-Policy` (CSP)
- **Severity**: Low
- **Description**: The response does not define a `Content-Security-Policy` header,
  omitting defense-in-depth against unauthorized content injection and cross-site
  scripting.
- **Remediation**: Deploy a Content Security Policy tailored to the application's
  resource requirements (e.g. `default-src 'self'`).

### 8. `WEB-HYGIENE-008`: Missing Clickjacking Protection
- **Severity**: Low
- **Description**: The response does not provide an `X-Frame-Options` header
  (`DENY` or `SAMEORIGIN`) or a CSP `frame-ancestors` directive, leaving the page
  potentially embeddable in malicious third-party iframes.
- **Remediation**: Send `X-Frame-Options: SAMEORIGIN` or CSP `frame-ancestors 'self'`.
