package duplicates

import (
	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Findings returns the site-wide duplicate findings. Each exact-duplicate and
// near-duplicate group is reported once, against its representative page.
func (r *Result) Findings() []findings.Finding {
	var out []findings.Finding
	for _, g := range r.Exact {
		out = append(out, findings.DuplicateExact.New(g.Representative, g.text()))
	}
	for _, g := range r.Near {
		out = append(out, findings.DuplicateNear.New(g.Representative, g.text()))
	}
	return out
}
