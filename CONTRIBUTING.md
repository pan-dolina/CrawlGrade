# Contributing to CrawlGrade

Thank you for your interest in contributing to CrawlGrade! CrawlGrade is an
open-source Go CLI tool for bounded, SSRF-safe technical SEO audits and
passive web hygiene checks.

## Code of Conduct

All contributors and participants are expected to adhere to our
[Code of Conduct](CODE_OF_CONDUCT.md).

## Development Setup

CrawlGrade requires Go 1.26 or later.

Clone the repository and verify the build and tests:

```sh
git clone https://github.com/pan-dolina/crawlgrade.git
cd crawlgrade
go test ./...
```

To build the binary:

```sh
go build -o bin/crawlgrade ./cmd/crawlgrade
```

> **Note**: Do not run `go build ./...` directly from the repository root, as
> Go will place a `crawlgrade` binary into the root directory. Always use
> `go build -o bin/crawlgrade ./cmd/crawlgrade` or `go vet ./...`.

## Core Architectural Invariants

When contributing code, please keep the following non-negotiable rules in mind:

1. **Hostile Input Boundary**: Everything fetched from an audited site is hostile
   input. Always bound loops, memory allocations, and body reading. Never panic
   on malformed input or edge cases.
2. **SSRF and Network Safety**: All network access MUST go through the `fetcher`
   package using the `netguard` dialer. Never introduce code paths that dial,
   resolve DNS, or follow redirects directly. Private and cloud-metadata addresses
   stay blocked by default.
3. **Deterministic, Offline Testing**: Tests must never access the public internet.
   Use `httptest`, `internal/testsite`, and the custom fake dialer/resolver
   patterns demonstrated in `internal/fetcher/ssrf_test.go`.
4. **Stable Public Contracts**: Finding IDs (e.g. `SEO-TITLE-001`, `WEB-HYGIENE-001`)
   are public contracts. Never rename, renumber, or reuse IDs. The JSON report
   uses `schema_version: "1"`.
5. **Standalone, Escaped HTML**: The generated HTML audit report is self-contained.
   It must not rely on external CDNs, analytics, remote scripts, or any active
   scripting derived from untrusted site data. All site-derived data must be
   strictly escaped to prevent stored XSS.

## Git and Commit Guidelines

- Use **Conventional Commits**:
  - `feat:` new feature or rule
  - `fix:` bug fix
  - `test:` test coverage or test fixtures
  - `perf:` performance improvement or benchmark
  - `docs:` documentation updates
  - `ci:` CI workflow changes
  - `build:` build and packaging changes
  - `security:` security controls or vulnerability mitigation
  - `chore:` maintenance tasks
- One logical change per commit. Ensure every commit builds cleanly and passes
  `go test ./...`.
- Real, incremental commits only. Never fake commit authorship or dates.

## Quality Gate

Before submitting a pull request, ensure the full quality gate passes locally:

```sh
gofmt -l cmd internal test
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
gosec -exclude-dir=.tools -exclude-dir=.gopath ./...
govulncheck ./...
go mod verify
```

## Security Disclosures

If you discover a potential security vulnerability (especially related to SSRF
bypass, parser panics, or HTML report escaping), please follow the instructions
in [SECURITY.md](SECURITY.md) to report it responsibly.
