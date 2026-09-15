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
