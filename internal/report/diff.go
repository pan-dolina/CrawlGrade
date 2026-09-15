package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Diff describes the difference between a baseline report and a current one.
// A finding is "new" when it appears in current but not baseline, and
// "resolved" when it appeared in baseline but not current.
type Diff struct {
	// New lists findings present in the current report but absent from the
	// baseline, sorted.
	New []findings.Finding
	// Resolved lists findings present in the baseline but absent from the
	// current report, sorted.
	Resolved []findings.Finding
	// AddedGroups lists group names present in current but not baseline.
	AddedGroups []string
	// RemovedGroups lists group names present in baseline but not current.
	RemovedGroups []string
}

// Equal reports whether the diff is empty: the two reports produced the same
// findings in the same groups.
func (d Diff) Equal() bool {
	return len(d.New) == 0 && len(d.Resolved) == 0 && len(d.AddedGroups) == 0 && len(d.RemovedGroups) == 0
}

// Key identifies a finding uniquely for comparison. Two findings with the same
// key are treated as the same observation across runs.
func Key(f findings.Finding) string {
	parts := []string{f.ID, f.URL}
	parts = append(parts, f.Evidence...)
	return strings.Join(parts, "\x00")
}

// Compare returns the difference between baseline and current. Both reports
// must share the same schema version; a mismatch is reported as an error.
func Compare(current, baseline *Report) (*Diff, error) {
	if baseline.Version != current.Version {
		return nil, fmt.Errorf("cannot diff reports with different schema versions (%s vs %s)", baseline.Version, current.Version)
	}
	diff := &Diff{}
	currentByKey := map[string]findings.Finding{}
	for _, f := range flatten(current) {
		currentByKey[Key(f)] = f
	}
	baselineByKey := map[string]findings.Finding{}
	for _, f := range flatten(baseline) {
		baselineByKey[Key(f)] = f
	}
	for k, f := range currentByKey {
		if _, ok := baselineByKey[k]; !ok {
			diff.New = append(diff.New, f)
		}
	}
	for k := range baselineByKey {
		if _, ok := currentByKey[k]; !ok {
			// Re-look up the finding to attach its metadata.
			for _, f := range flatten(baseline) {
				if Key(f) == k {
					diff.Resolved = append(diff.Resolved, f)
					break
				}
			}
		}
	}
	diff.AddedGroups = missingGroups(current.Groups, baseline.Groups)
	diff.RemovedGroups = missingGroups(baseline.Groups, current.Groups)
	findings.Sort(diff.New)
	findings.Sort(diff.Resolved)
	sort.Strings(diff.AddedGroups)
	sort.Strings(diff.RemovedGroups)
	return diff, nil
}

// flatten returns every finding in a report, grouped then sorted, so the
// comparison is order-independent.
func flatten(rep *Report) []findings.Finding {
	var out []findings.Finding
	for _, fs := range rep.Groups {
		out = append(out, fs...)
	}
	findings.Sort(out)
	return out
}

// missingGroups returns the group names present in present but absent in
// absent, sorted.
func missingGroups(present, absent map[string][]findings.Finding) []string {
	var out []string
	for name := range present {
		if _, ok := absent[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// RenderTerminal writes the diff to w as terminal text.
func (d *Diff) RenderTerminal(w *strings.Builder) {
	if d.Equal() {
		w.WriteString("No changes since the baseline.\n")
		return
	}
	if len(d.New) > 0 {
		w.WriteString(fmt.Sprintf("\n%d new finding(s):\n", len(d.New)))
		for _, f := range d.New {
			writeFinding(w, f)
		}
	}
	if len(d.Resolved) > 0 {
		w.WriteString(fmt.Sprintf("\n%d resolved finding(s):\n", len(d.Resolved)))
		for _, f := range d.Resolved {
			fmt.Fprintf(w, "  resolved [%s] %s%s\n", f.Severity, f.Title, urlLine(f.URL))
		}
	}
	if len(d.AddedGroups) > 0 {
		w.WriteString(fmt.Sprintf("\nNew categories: %s\n", strings.Join(d.AddedGroups, ", ")))
	}
	if len(d.RemovedGroups) > 0 {
		w.WriteString(fmt.Sprintf("\nRemoved categories: %s\n", strings.Join(d.RemovedGroups, ", ")))
	}
}

// urlLine returns a URL suffix for resolved findings.
func urlLine(url string) string {
	if url == "" {
		return ""
	}
	return " (" + url + ")"
}

// Load parses a saved report from bytes. The bytes must be JSON produced by
// RenderJSON; the terminal and HTML formats are not self-describing and cannot
// be loaded as a baseline.
func Load(data []byte) (*Report, error) {
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}
