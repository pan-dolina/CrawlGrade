package htmlcheck

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Length heuristics. Search engines truncate by pixel width, not by
// characters, so these are approximations and the findings say so.
const (
	TitleMinRunes       = 15
	TitleMaxRunes       = 65
	DescriptionMinRunes = 50
	DescriptionMaxRunes = 160
)

// MetadataFindings returns title and description findings for one page.
func (p *Page) MetadataFindings() []findings.Finding {
	var out []findings.Finding
	switch {
	case len(p.Titles) == 0:
		out = append(out, findings.TitleMissing.New(p.URL))
	case p.Title() == "":
		out = append(out, findings.TitleEmpty.New(p.URL))
	default:
		if n := utf8.RuneCountInString(p.Title()); n < TitleMinRunes {
			out = append(out, findings.TitleShort.New(p.URL, fmt.Sprintf("%d characters: %q", n, p.Title())))
		} else if n > TitleMaxRunes {
			out = append(out, findings.TitleLong.New(p.URL, fmt.Sprintf("%d characters: %q", n, p.Title())))
		}
	}
	if len(p.Titles) > 1 {
		out = append(out, findings.TitleMultiple.New(p.URL, quoteAll(p.Titles)...))
	}

	switch {
	case len(p.Descriptions) == 0:
		out = append(out, findings.DescriptionMissing.New(p.URL))
	case p.Description() == "":
		out = append(out, findings.DescriptionEmpty.New(p.URL))
	default:
		n := utf8.RuneCountInString(p.Description())
		if n < DescriptionMinRunes || n > DescriptionMaxRunes {
			out = append(out, findings.DescriptionLength.New(p.URL, fmt.Sprintf("%d characters (approximate guideline %d-%d)", n, DescriptionMinRunes, DescriptionMaxRunes)))
		}
	}
	if len(p.Descriptions) > 1 {
		out = append(out, findings.DescriptionMultiple.New(p.URL, quoteAll(p.Descriptions)...))
	}
	return out
}

func quoteAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}

// DuplicateMetadataFindings reports titles and descriptions shared by more
// than one page. Pages are compared by exact text after whitespace
// collapsing; empty values are reported by the per-page checks instead.
func DuplicateMetadataFindings(pages []*Page) []findings.Finding {
	var out []findings.Finding
	out = append(out, duplicates(pages, (*Page).Title, findings.TitleDuplicate)...)
	out = append(out, duplicates(pages, (*Page).Description, findings.DescriptionDuplicate)...)
	return out
}

func duplicates(pages []*Page, value func(*Page) string, rule findings.Rule) []findings.Finding {
	groups := map[string][]string{}
	for _, p := range pages {
		if v := value(p); v != "" {
			groups[v] = append(groups[v], p.URL)
		}
	}
	var out []findings.Finding
	for v, urls := range groups {
		if len(urls) < 2 {
			continue
		}
		slices.Sort(urls)
		for _, u := range urls {
			ev := []string{fmt.Sprintf("%q", v)}
			for _, other := range urls {
				if other != u {
					ev = append(ev, "also on "+other)
				}
			}
			out = append(out, rule.New(u, ev...))
		}
	}
	findings.Sort(out)
	return out
}
