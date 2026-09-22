# Release and verification

Release automation builds Linux amd64/arm64, macOS amd64/arm64 and Windows
amd64 archives, checks reproducibility, smoke-tests extracted binaries,
generates SPDX and CycloneDX SBOMs, and signs artifacts with Sigstore keyless
signing. Actions are pinned to full commit SHAs. Publishing uses a `release`
environment: configure required reviewers and tag restrictions in GitHub.

Local preparation (Go toolchain from `go.mod`; Syft required):

```sh
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
gosec -quiet -exclude-dir=.tools -exclude-dir=.gopath ./...
govulncheck ./...
osv-scanner scan source --lockfile=go.mod
go mod verify
VERSION=v0.1.0 scripts/build-release.sh
VERSION=v0.1.0 scripts/verify-reproducible.sh
scripts/sbom.sh sbom
```

The output directory must be empty. Build metadata and archive timestamps use
the source commit, not fabricated release dates. Rebuilding requires the same
source and toolchain. A dirty-tree build is useful for local verification but
is not a release reproducible from a published commit.

For a native extracted binary:

```sh
CRAWLGRADE_BIN=/absolute/path/crawlgrade CRAWLGRADE_EXPECT_VERSION=v0.1.0 \
  go test -tags smoke -count=1 ./test/smoke
```

Once a signed release exists, verify downloaded checksums and signatures:

```sh
sha256sum -c SHA256SUMS
cosign verify-blob --bundle SHA256SUMS.sigstore.json \
  --certificate-identity 'https://github.com/pan-dolina/CrawlGrade/.github/workflows/release.yml@refs/tags/v0.1.0' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com SHA256SUMS
gh attestation verify crawlgrade_0.1.0_linux_amd64.tar.gz --repo pan-dolina/CrawlGrade
```

Hosted matrix results, keyless signatures and provenance can only be verified
after the workflow runs on GitHub. Local tests do not establish those results.
