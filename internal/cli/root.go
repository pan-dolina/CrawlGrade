// Package cli implements the crawlgrade command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"io"
	"os"
	"time"

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
crawlability, metadata, content, structured data and internal linking.
Passive web hygiene is reported separately.

CrawlGrade does not predict search engine rankings.

Exit codes:
  0  audit completed, no finding reached the --fail-on threshold
  1  at least one finding reached the --fail-on threshold
  2  invalid arguments
  3  target not auditable: blocked by the network policy or robots.txt
  4  start URL could not be fetched, or the run was interrupted
  5  internal error`

func (a *App) newRootCommand() *cobra.Command {
	var (
		format   string
		failOn   string
		baseline string
		diffMode bool
		noColor  bool
		private  bool
		maxPages int
		maxDepth int
		conc     int
		verbose  bool
	)
	root := &cobra.Command{
		Use:           "crawlgrade URL [flags]",
		Short:         "Technical SEO audit and passive web hygiene checks for a website",
		Long:          rootLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runAudit(a, cmd, args[0], format, failOn, baseline, diffMode, noColor, private, verbose, maxPages, maxDepth, conc)
		},
	}
	flags(root, &format, &failOn, &baseline, &diffMode, &noColor, &maxPages, &maxDepth, &conc)
	root.Flags().BoolVar(&private, "allow-private", false, "allow loopback, private and link-local destinations (cloud metadata addresses stay blocked)")
	root.SetOut(a.Stdout)
	root.SetErr(a.Stderr)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return withCode(ExitUsage, err)
	})
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(a.newVersionCommand(), a.newDiffCommand())
	root.Flags().Bool("json", false, "write a JSON report")
	root.Flags().String("output", "", "write the report to this file")
	root.Flags().String("keywords", "", "comma-separated terms to include in term scores")
	root.Flags().Duration("max-duration", crawler.DefaultMaxDuration, "maximum total audit duration")
	root.Flags().Duration("timeout", 15*time.Second, "timeout per fetch including redirects")
	root.Flags().String("user-agent", "CrawlGrade", "HTTP user agent (robots policy uses crawlgrade)")
	root.Flags().Float64("requests-per-second", 5, "maximum request starts per second, including redirects")
	root.Flags().Bool("check-external-links", false, "check external link status with bounded HEAD requests")
	root.Flags().BoolVar(&verbose, "verbose", false, "show full finding evidence, recommendations and page metrics")
	root.Flags().Bool("quick-wins", false, "HTML only: show the summary and prioritized quick wins")
	return root
}

func flags(root *cobra.Command, format, failOn, baseline *string, diffMode, noColor *bool, maxPages, maxDepth, conc *int) {
	root.Flags().StringVar(format, "format", "terminal", "report format: terminal, json or html")
	root.Flags().StringVar(failOn, "fail-on", "", "exit 1 if a finding of this severity or higher is found: info, low, medium, high, critical")
	root.Flags().StringVar(baseline, "baseline", "", "path to a previous JSON report to diff against")
	root.Flags().BoolVar(diffMode, "diff", false, "print only the difference from the baseline report")
	root.Flags().BoolVar(noColor, "no-color", false, "disable ANSI colour in the terminal report")
	root.Flags().IntVar(maxPages, "max-pages", crawler.DefaultMaxPages, "maximum number of pages to crawl")
	root.Flags().IntVar(maxDepth, "max-depth", crawler.DefaultMaxDepth, "maximum link depth from the start URL")
	root.Flags().IntVar(conc, "concurrency", crawler.DefaultConcurrency, "maximum concurrent requests")
}
