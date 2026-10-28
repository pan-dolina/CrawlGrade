package report

import (
	"fmt"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/terms"
)

type PageMetric struct {
	URL       string        `json:"url"`
	Status    int           `json:"status"`
	Depth     int           `json:"depth"`
	Indexable bool          `json:"indexable"`
	Inbound   int           `json:"inbound"`
	Outbound  int           `json:"outbound"`
	Terms     []terms.Score `json:"terms,omitempty"`
}

// Scores describe observed technical findings, never search rankings.
type Scores struct {
	SEO        int `json:"seo"`
	WebHygiene int `json:"web_hygiene"`
}

// BuildHighlights populates rep.Strengths and rep.Weaknesses from the
// report's current terms, scores, page metrics, and findings.
func (rep *Report) BuildHighlights() {
	rep.Strengths, rep.Weaknesses = rep.Highlights()
}

// Highlights computes a deterministic summary of strengths and weaknesses,
// including the strongest and weakest keywords, score health, page status
// health, and category findings.
func (rep *Report) Highlights() (strengths, weaknesses []string) {
	if len(rep.Terms) > 0 {
		var best, worst, missing []string
		bestSet := map[string]bool{}
		for _, t := range rep.Terms {
			if t.Strength <= 0 {
				missing = append(missing, t.Term)
				continue
			}
			if !terms.IsKeyword(t.Term) {
				continue
			}
			if len(best) < 5 && (t.Strength >= 50 || len(best) == 0) {
				best = append(best, fmt.Sprintf("%s (%.1f)", t.Term, t.Strength))
				bestSet[t.Term] = true
			}
		}
		if len(best) > 0 {
			strengths = append(strengths, "Best keywords: "+strings.Join(best, ", "))
		}
		if len(missing) > 0 {
			weaknesses = append(weaknesses, "Missing target keywords (0.0): "+strings.Join(missing, ", "))
		}
		for _, t := range rep.WeakTerms {
			if len(worst) >= 5 {
				break
			}
			if t.Strength <= 0 || bestSet[t.Term] || !terms.IsKeyword(t.Term) {
				continue
			}
			worst = append(worst, fmt.Sprintf("%s (%.1f)", t.Term, t.Strength))
		}
		if len(worst) == 0 {
			topStrength := rep.Terms[0].Strength
			for i := len(rep.Terms) - 1; i >= 0 && len(worst) < 5; i-- {
				t := rep.Terms[i]
				if t.Strength <= 0 || bestSet[t.Term] || !terms.IsKeyword(t.Term) {
					continue
				}
				if t.Strength < 50 || t.Strength < topStrength {
					worst = append(worst, fmt.Sprintf("%s (%.1f)", t.Term, t.Strength))
				}
			}
		}
		if len(worst) > 0 {
			weaknesses = append(weaknesses, "Weakest keywords: "+strings.Join(worst, ", "))
		}
	} else if rep.Summary.Pages > 0 {
		weaknesses = append(weaknesses, "No significant keywords extracted from crawled pages")
	}

	if rep.Scores != nil {
		if rep.Scores.SEO >= 80 {
			strengths = append(strengths, fmt.Sprintf("Strong technical SEO score (%d/100)", rep.Scores.SEO))
		} else {
			weaknesses = append(weaknesses, fmt.Sprintf("Low technical SEO score (%d/100)", rep.Scores.SEO))
		}
		if rep.Scores.WebHygiene >= 80 {
			strengths = append(strengths, fmt.Sprintf("Strong passive web hygiene score (%d/100)", rep.Scores.WebHygiene))
		} else {
			weaknesses = append(weaknesses, fmt.Sprintf("Low passive web hygiene score (%d/100)", rep.Scores.WebHygiene))
		}
	}

	if len(rep.Pages) > 0 {
		ok2xx, errStatus, indexable, unlinked := 0, 0, 0, 0
		bestURL := ""
		bestInbound := 0
		for i, p := range rep.Pages {
			if p.Status >= 200 && p.Status < 300 {
				ok2xx++
			} else if p.Status >= 400 || p.Status == 0 {
				errStatus++
			}
			if p.Indexable {
				indexable++
				if i > 0 && p.URL != rep.Summary.StartURL && p.Inbound == 0 {
					unlinked++
				}
			}
			if p.Indexable && p.Inbound > bestInbound {
				bestInbound = p.Inbound
				bestURL = p.URL
			}
		}
		if errStatus == 0 && ok2xx > 0 {
			strengths = append(strengths, fmt.Sprintf("All %d crawled page(s) returned healthy HTTP 2xx status", len(rep.Pages)))
		} else if errStatus > 0 {
			weaknesses = append(weaknesses, fmt.Sprintf("%d of %d crawled page(s) failed or returned HTTP 4xx/5xx status", errStatus, len(rep.Pages)))
		}
		if indexable > 0 {
			strengths = append(strengths, fmt.Sprintf("%d of %d crawled page(s) are indexable", indexable, len(rep.Pages)))
		} else {
			weaknesses = append(weaknesses, "No crawled pages are indexable")
		}
		if bestInbound > 0 {
			strengths = append(strengths, fmt.Sprintf("Most linked internal page: %s (%d inbound links)", bestURL, bestInbound))
		}
		if unlinked > 0 {
			weaknesses = append(weaknesses, fmt.Sprintf("%d indexable page(s) have no inbound internal links", unlinked))
		}
	}

	crit := rep.Summary.Findings["critical"]
	high := rep.Summary.Findings["high"]
	med := rep.Summary.Findings["medium"]
	if crit == 0 && high == 0 {
		strengths = append(strengths, "No critical or high-severity findings detected")
	} else {
		weaknesses = append(weaknesses, fmt.Sprintf("High-priority findings detected: %d critical, %d high", crit, high))
	}

	var cleanCats, problemCats []string
	for _, g := range orderedGroups {
		fs := rep.Groups[g]
		name := findings.Category(g).Name()
		actionable := 0
		for _, f := range fs {
			if f.Severity >= findings.SeverityMedium {
				actionable++
			}
		}
		if len(fs) == 0 {
			cleanCats = append(cleanCats, name)
		} else if actionable > 0 {
			problemCats = append(problemCats, fmt.Sprintf("%s (%d)", name, len(fs)))
		}
	}
	if len(cleanCats) > 0 && (rep.Summary.Pages > 0 || len(rep.CategoryGroups()) > 0) {
		strengths = append(strengths, "Clean categories: "+strings.Join(cleanCats, ", "))
	}
	if len(problemCats) > 0 {
		weaknesses = append(weaknesses, "Categories needing attention: "+strings.Join(problemCats, ", "))
	} else if crit == 0 && high == 0 && med == 0 && len(rep.CategoryGroups()) > 0 {
		strengths = append(strengths, "No medium- or higher-severity issues in any category")
	}

	seenRule := map[string]bool{}
	var topIssues []string
	for _, sev := range []findings.Severity{findings.SeverityCritical, findings.SeverityHigh, findings.SeverityMedium} {
		for _, g := range orderedGroups {
			for _, f := range rep.Groups[g] {
				if f.Severity != sev || seenRule[f.ID] {
					continue
				}
				seenRule[f.ID] = true
				topIssues = append(topIssues, fmt.Sprintf("[%s] %s (%s)", f.ID, f.Title, f.Severity))
				if len(topIssues) == 3 {
					break
				}
			}
			if len(topIssues) == 3 {
				break
			}
		}
		if len(topIssues) == 3 {
			break
		}
	}
	for _, issue := range topIssues {
		weaknesses = append(weaknesses, "Key issue: "+issue)
	}

	return strengths, weaknesses
}
