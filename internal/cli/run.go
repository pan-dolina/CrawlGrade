package cli

import (
	"bytes"
	"io"
	"net/url"
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
func runAudit(a *App, cmd *cobra.Command, start, format, failOn, baseline string, diffMode, noColor, allowPrivate bool, maxPages, maxDepth, conc int) error {
	u, err := url.Parse(start)
	if err != nil {
		return usageErrorf("invalid start URL %q: %v", start, err)
	}

	dialer := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: allowPrivate}}
	f := fetcher.New(fetcher.Options{DialContext: dialer.DialContext})

	res := audit.Run(cmd.Context(), f, u, audit.Options{
		MaxPages:     maxPages,
		MaxDepth:     maxDepth,
		Concurrency:  conc,
		AllowPrivate: allowPrivate,
	})
	rep := res.Report

	out, err := renderReport(a.Stdout, rep, format, baseline, diffMode)
	if err != nil {
		return err
	}
	if _, err := cmd.OutOrStdout().Write(out); err != nil {
		return err
	}
	return exitFor(rep, failOn)
}

// renderReport renders the report, or its diff against baseline when requested.
func renderReport(w io.Writer, rep *report.Report, format, baseline string, diffMode bool) ([]byte, error) {
	if baseline != "" {
		return renderBaseline(w, rep, baseline, diffMode, format)
	}
	var buf bytes.Buffer
	if err := rep.Render(&buf, mustFormat(format)); err != nil {
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
		var buf strings.Builder
		diff.RenderTerminal(&buf)
		return []byte(buf.String()), nil
	}
	// Full report mode: render the current report as usual.
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
	//nolint:gosec // G304: path is user-supplied by the --baseline flag.
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
	if reportMaxSeverity(rep) >= threshold {
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
