// Package cli implements the crawlgrade command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// App wires the command tree to its I/O streams.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	// Getenv is used for NO_COLOR and similar environment lookups.
	Getenv func(string) string
}

// NewApp returns an App bound to the process streams.
func NewApp() *App {
	return &App{Stdout: os.Stdout, Stderr: os.Stderr, Getenv: os.Getenv}
}

// Execute runs the CLI with the given arguments and returns the exit code.
func (a *App) Execute(ctx context.Context, args []string) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(a.Stderr, "crawlgrade: internal error: %v\n", r)
			code = ExitInternal
		}
	}()
	root := a.newRootCommand()
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	code = ExitCode(err)
	if err != nil {
		var ee *exitError
		if !errors.As(err, &ee) || ee.err != nil {
			fmt.Fprintf(a.Stderr, "crawlgrade: %v\n", err)
		}
		if code == ExitUsage {
			fmt.Fprintln(a.Stderr, "Run 'crawlgrade --help' for usage.")
		}
	}
	return code
}

const rootLong = `CrawlGrade crawls a website within strict safety limits and audits its
technical SEO: crawlability, metadata, content, structured data and internal
linking. Passive web hygiene is reported separately.

CrawlGrade does not predict search engine rankings.

Exit codes:
  0  audit completed, no finding reached the --fail-on threshold
  1  at least one finding reached the --fail-on threshold
  2  invalid arguments
  3  target not auditable: blocked by the network policy or robots.txt
  4  start URL could not be fetched, or the run was interrupted
  5  internal error`

func (a *App) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "crawlgrade URL [flags]",
		Short:         "Technical SEO audit and passive web hygiene checks for a website",
		Long:          rootLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return usageErrorf("auditing is not available in this build yet")
		},
	}
	root.SetOut(a.Stdout)
	root.SetErr(a.Stderr)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return withCode(ExitUsage, err)
	})
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(a.newVersionCommand())
	return root
}
