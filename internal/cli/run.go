package cli

import (
	"bytes"
	"encoding/json"
	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/urlnorm"
	"io"
	"math"
	"os"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/audit"
	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
	"github.com/pan-dolina/crawlgrade/internal/report"
	"github.com/spf13/cobra"
)

// runAudit crawls start and writes the report in the requested format.
func runAudit(a *App, cmd *cobra.Command, start, format, failOn, baseline string, diffMode, noColor, allowPrivate, verbose bool, maxPages, maxDepth, conc int) error {
	u, err := urlnorm.Parse(start)
	if err != nil {
		return usageErrorf("invalid start URL %q: %v", start, err)
	}

	if _, err := report.ParseFormat(format); err != nil {
		return usageErrorf("%v", err)
	}
	if failOn != "" {
		if _, err := findings.ParseSeverity(failOn); err != nil {
			return usageErrorf("--fail-on: %v", err)
		}
	}
	if maxPages < 1 || maxPages > 10000 || maxDepth < 0 || maxDepth > 100 || conc < 1 || conc > crawler.MaxConcurrency {
		return usageErrorf("limits: pages 1..10000, depth 0..100, concurrency 1..%d", crawler.MaxConcurrency)
	}
	if diffMode && format == "html" {
		return usageErrorf("--diff supports terminal or json output")
	}
	if diffMode && baseline == "" {
		return usageErrorf("--diff requires --baseline")
	}
	if baseline != "" {
		data, err := readBaseline(baseline)
		if err != nil {
			return err
		}
		if _, err := report.Load(data); err != nil {
			return usageErrorf("invalid baseline: %v", err)
		}
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	quickWins, _ := cmd.Flags().GetBool("quick-wins")
	if quickWins && format != "html" {
		return usageErrorf("--quick-wins requires --format html")
	}
	if asJSON {
		format = "json"
	}
	output, _ := cmd.Flags().GetString("output")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	duration, _ := cmd.Flags().GetDuration("max-duration")
	if duration <= 0 {
		return usageErrorf("--max-duration must be positive")
	}
	ua, _ := cmd.Flags().GetString("user-agent")
	rps, _ := cmd.Flags().GetFloat64("requests-per-second")
	if timeout <= 0 || rps <= 0 || math.IsNaN(rps) || math.IsInf(rps, 0) || strings.TrimSpace(ua) == "" || strings.ContainsAny(ua, "\r\n") {
		return usageErrorf("timeout and request rate must be positive; user agent must be a nonempty single line")
	}
	dialer := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: allowPrivate}}
	f := fetcher.New(fetcher.Options{DialContext: dialer.DialContext, Timeout: timeout, UserAgent: ua, Wait: crawler.NewLimiter(rps).Wait})
	defer f.Close()

	keywords, _ := cmd.Flags().GetString("keywords")
	external, _ := cmd.Flags().GetBool("check-external-links")
	res := audit.Run(cmd.Context(), f, u, audit.Options{
		MaxPages:           maxPages,
		MaxDuration:        duration,
		MaxDepth:           maxDepth,
		Concurrency:        conc,
		AllowPrivate:       allowPrivate,
		Keywords:           strings.Split(keywords, ","),
		CheckExternalLinks: external,
	})
	rep := res.Report

	out, err := renderReport(a.Stdout, rep, format, baseline, diffMode, verbose, quickWins)
	if err != nil {
		return err
	}
	if output != "" {
		// #nosec G306 -- reports may contain private site information.
		if err := os.WriteFile(output, out, 0600); err != nil {
			return withCode(ExitInternal, err)
		}
	} else if _, err := cmd.OutOrStdout().Write(out); err != nil {
		return withCode(ExitInternal, err)
	}
	if cmd.Context().Err() != nil || res.Err != nil {
		return silentExit(ExitIncomplete)
	}
	if res.Crawl.StopReason == crawler.StopStartDisallowed {
		return silentExit(ExitBlocked)
	}
	if len(res.Crawl.Pages) == 0 {
		return silentExit(ExitIncomplete)
	}
	first := res.Crawl.Pages[0]
	if first.ErrorKind == crawler.ErrorBlocked {
		return silentExit(ExitBlocked)
	}
	if first.Err != nil || first.Response == nil || first.Response.Status >= 400 || res.Crawl.StopReason == crawler.StopInterrupted || res.Crawl.StopReason == crawler.StopMaxDuration {
		return silentExit(ExitIncomplete)
	}
	return exitFor(rep, failOn)
}

// renderReport renders the report, or its diff against baseline when requested.
func renderReport(w io.Writer, rep *report.Report, format, baseline string, diffMode, verbose, quickWins bool) ([]byte, error) {
	if baseline != "" {
		return renderBaseline(w, rep, baseline, diffMode, format)
	}
	var buf bytes.Buffer
	if quickWins {
		if err := report.RenderHTMLQuickWins(&buf, rep); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	if mustFormat(format) == report.FormatTerminal && verbose {
		if err := report.RenderTerminalDetailed(&buf, rep); err != nil {
			return nil, err
		}
	} else if err := rep.Render(&buf, mustFormat(format)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mustFormat parses the format, defaulting to terminal on error.
func mustFormat(format string) report.Format {
	f, err := report.ParseFormat(format)
	if err != nil {
		f = report.FormatTerminal
	}
	return f
}

// renderBaseline loads the saved report, diffs it against the current one and
// renders the result.
func renderBaseline(w io.Writer, current *report.Report, baseline string, diffMode bool, format string) ([]byte, error) {
	data, err := readBaseline(baseline)
	if err != nil {
		return nil, err
	}
	base, err := report.Load(data)
	if err != nil {
		return nil, err
	}
	diff, err := report.Compare(current, base)
	if err != nil {
		return nil, err
	}
	if diffMode {
		if format == "json" {
			return json.MarshalIndent(diff, "", "  ")
		}
		var buf strings.Builder
		diff.RenderTerminal(&buf)
		return []byte(report.TerminalText(buf.String())), nil
	}
	// Full report mode retains the delta alongside current observations.
	current.Baseline = diff
	var out bytes.Buffer
	if err := current.Render(&out, report.Format(format)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// readBaseline reads the baseline file, returning a helpful error if missing.
func readBaseline(path string) ([]byte, error) {
	// Reading the baseline is the whole point of --baseline: the user
	// supplies the path, so this is intended file inclusion, not a traversal.
	// #nosec G304 -- baseline input is an explicit user-selected local file.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, usageErrorf("cannot read baseline %q: %v", path, err)
	}
	return data, nil
}

// exitFor returns a terminal error when a finding reached the --fail-on
// threshold.
func exitFor(rep *report.Report, failOn string) error {
	if failOn == "" {
		return nil
	}
	threshold, err := findings.ParseSeverity(failOn)
	if err != nil {
		return usageErrorf("--fail-on: %v", err)
	}
	if rep.Summary.Highest != "" && reportMaxSeverity(rep) >= threshold {
		return silentExit(ExitFindings)
	}
	return nil
}

// reportMaxSeverity returns the highest severity across all groups.
func reportMaxSeverity(rep *report.Report) findings.Severity {
	var worst findings.Severity
	for _, group := range rep.CategoryGroups() {
		if s := findings.Max(rep.Findings(group)); s > worst {
			worst = s
		}
	}
	return worst
}
