package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// renderHTML writes a standalone HTML report to w. The document is fully
// self-contained: it embeds its styling, loads nothing from the network, and
// contains no script derived from the audited site. All dynamic values are
// passed through html/template, which escapes them for the context they are
// rendered in.
func renderHTML(w io.Writer, rep *Report) error { return renderHTMLMode(w, rep, false) }

// RenderHTMLQuickWins writes a focused standalone report containing the
// summary and a short prioritized action list.
func RenderHTMLQuickWins(w io.Writer, rep *Report) error { return renderHTMLMode(w, rep, true) }

func renderHTMLMode(w io.Writer, rep *Report, quickOnly bool) error {
	data := htmlModel{
		Report:     rep,
		StartURL:   rep.Summary.StartURL,
		Pages:      rep.Summary.Pages,
		Counts:     summaryCounts(rep.Summary.Findings),
		Highest:    rep.Summary.Highest,
		Stop:       rep.Summary.StopReason,
		Skipped:    skippedRows(rep.Summary.Skipped),
		Groups:     reportGroups(rep),
		QuickWins:  quickWins(rep),
		Severities: severitySections(rep),
		QuickOnly:  quickOnly,
		LLMPrompt:  buildLLMPrompt(rep, rep.Summary.StartURL, quickOnly),
	}
	t, err := loadHTMLTemplate()
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

// htmlModel is the template model. Every string field is escaped by
// html/template for the context it is rendered in.
type htmlModel struct {
	Report     *Report
	StartURL   string
	Pages      int
	Counts     severityCounts
	Highest    string
	Stop       string
	Skipped    []skipRow
	Groups     []groupModel
	QuickWins  []findingModel
	Severities []severitySection
	QuickOnly  bool
	LLMPrompt  string
}

type severitySection struct {
	Title    string
	Severity string
	Count    int
	Findings []findingModel
}

func severitySections(rep *Report) []severitySection {
	var out []severitySection
	for _, sev := range findings.Severities() {
		section := severitySection{Title: sev.String(), Severity: sev.String()}
		for _, group := range orderedGroups {
			catName := findings.Category(group).Name()
			for _, f := range rep.Groups[group] {
				if f.Severity != sev {
					continue
				}
				section.Findings = append(section.Findings, findingModel{
					ID:             f.ID,
					Severity:       f.Severity.String(),
					Title:          f.Title,
					URL:            f.URL,
					Evidence:       f.Evidence,
					Rec:            f.Recommendation,
					Category:       catName,
					LLMInstruction: singleFindingInstruction(f.ID, f.Title, f.Severity.String(), catName, f.URL, f.Evidence, f.Recommendation),
				})
			}
		}
		section.Count = len(section.Findings)
		if section.Count > 0 {
			out = append(out, section)
		}
	}
	return out
}

func quickWins(rep *Report) []findingModel {
	var out []findingModel
	seen := map[string]bool{}
	for _, sev := range []findings.Severity{findings.SeverityCritical, findings.SeverityHigh, findings.SeverityMedium} {
		for _, group := range orderedGroups {
			catName := findings.Category(group).Name()
			for _, f := range rep.Groups[group] {
				if f.Severity != sev || seen[f.ID] {
					continue
				}
				seen[f.ID] = true
				evidence := f.Evidence[:min(1, len(f.Evidence))]
				out = append(out, findingModel{
					ID:             f.ID,
					Severity:       f.Severity.String(),
					Title:          f.Title,
					URL:            f.URL,
					Evidence:       evidence,
					Rec:            f.Recommendation,
					Category:       catName,
					LLMInstruction: singleFindingInstruction(f.ID, f.Title, f.Severity.String(), catName, f.URL, evidence, f.Recommendation),
				})
				if len(out) == 8 {
					return out
				}
			}
		}
	}
	return out
}

// severityCounts holds the per-severity tallies for the summary banner.
type severityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Info     int
}

// summaryCounts copies the global tally into a fixed-shape struct so the
// template can render every level even when it is zero.
func summaryCounts(m map[string]int) severityCounts {
	return severityCounts{
		Critical: m["critical"],
		High:     m["high"],
		Medium:   m["medium"],
		Low:      m["low"],
		Info:     m["info"],
	}
}

// skipRow is one skipped-URL reason and its count.
type skipRow struct {
	Reason string
	Count  int
}

// skippedRows converts the skipped-count map into a sorted slice for the
// template.
func skippedRows(m map[string]int) []skipRow {
	if len(m) == 0 {
		return nil
	}
	rows := make([]skipRow, 0, len(m))
	for reason, n := range m {
		rows = append(rows, skipRow{Reason: reason, Count: n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Reason < rows[j].Reason })
	return rows
}

// groupModel is one analysis group and its findings for the template.
type groupModel struct {
	Title    string
	Severity string
	Findings []findingModel
}

// findingModel is a template-safe view of a finding.
type findingModel struct {
	ID             string
	Severity       string
	Title          string
	URL            string
	Evidence       []string
	Rec            string
	Category       string
	LLMInstruction string
}

// reportGroups builds the ordered, non-empty groups for the template.
func reportGroups(rep *Report) []groupModel {
	var groups []groupModel
	for _, group := range orderedGroups {
		fs := rep.Groups[group]
		if len(fs) == 0 {
			continue
		}
		catName := findingsCategoryName(group)
		gm := groupModel{Title: catName, Severity: mostSevereSeverity(fs)}
		for _, f := range fs {
			gm.Findings = append(gm.Findings, findingModel{
				ID:             f.ID,
				Severity:       f.Severity.String(),
				Title:          f.Title,
				URL:            f.URL,
				Evidence:       f.Evidence,
				Rec:            f.Recommendation,
				Category:       catName,
				LLMInstruction: singleFindingInstruction(f.ID, f.Title, f.Severity.String(), catName, f.URL, f.Evidence, f.Recommendation),
			})
		}
		groups = append(groups, gm)
	}
	return groups
}

// mostSevereSeverity returns the most severe finding's name in the group.
func mostSevereSeverity(fs []findings.Finding) string {
	var worst findings.Severity
	for _, f := range fs {
		if f.Severity > worst {
			worst = f.Severity
		}
	}
	return worst.String()
}

// singleFindingInstruction formats a tailored prompt for an LLM to resolve
// an individual finding on a page.
func singleFindingInstruction(id, title, severity, category, url string, evidence []string, rec string) string {
	var b strings.Builder
	b.WriteString("Działaj jako ekspert web developmentu i SEO. Napraw problem wykryty na stronie:\n\n")
	fmt.Fprintf(&b, "- Identyfikator reguły: %s\n", id)
	fmt.Fprintf(&b, "- Poziom istotności: %s\n", strings.ToUpper(severity))
	fmt.Fprintf(&b, "- Nazwa błędu: %s\n", title)
	if category != "" {
		fmt.Fprintf(&b, "- Kategoria: %s\n", category)
	}
	if url != "" {
		fmt.Fprintf(&b, "- Adres URL: %s\n", url)
	}
	for _, ev := range evidence {
		fmt.Fprintf(&b, "- Obserwacja audytu: %s\n", ev)
	}
	if rec != "" {
		fmt.Fprintf(&b, "- Zalecana akcja naprawcza: %s\n", rec)
	}
	b.WriteString("\nZadanie dla Ciebie:\n")
	b.WriteString("1. Przedstaw gotowy, poprawny kod źródłowy (HTML, nagłówki HTTP, Schema.org lub reguły serwera), który rozwiązuje powyższy błąd.\n")
	b.WriteString("2. Wyjaśnij zwięźle, jak bezpiecznie wdrożyć tę zmianę i jak przetestować jej poprawność.")
	return b.String()
}

// buildLLMPrompt compiles an actionable prompt for an LLM to address the
// findings discovered during the audit.
func buildLLMPrompt(rep *Report, startURL string, quickOnly bool) string {
	type item struct {
		ID       string
		Severity string
		Title    string
		URL      string
		Evidence []string
		Rec      string
		Category string
	}
	var items []item
	if quickOnly {
		for _, q := range quickWins(rep) {
			items = append(items, item{
				ID:       q.ID,
				Severity: q.Severity,
				Title:    q.Title,
				URL:      q.URL,
				Evidence: q.Evidence,
				Rec:      q.Rec,
				Category: q.Category,
			})
		}
	} else {
		for _, sev := range findings.Severities() {
			for _, group := range orderedGroups {
				for _, f := range rep.Groups[group] {
					if f.Severity != sev {
						continue
					}
					items = append(items, item{
						ID:       f.ID,
						Severity: f.Severity.String(),
						Title:    f.Title,
						URL:      f.URL,
						Evidence: f.Evidence,
						Rec:      f.Recommendation,
						Category: findings.Category(group).Name(),
					})
				}
			}
		}
	}
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Działaj jako senior web developer i ekspert technicznego SEO.\n")
	if startURL != "" {
		fmt.Fprintf(&b, "Oto zestawienie problemów wykrytych przez audyt CrawlGrade dla witryny %s.\n\n", startURL)
	} else {
		b.WriteString("Oto zestawienie problemów wykrytych przez audyt CrawlGrade.\n\n")
	}
	b.WriteString("LISTA WYKRYTYCH BŁĘDÓW DO NAPRAWY:\n\n")

	for i, it := range items {
		fmt.Fprintf(&b, "%d. [%s] [%s] %s", i+1, strings.ToUpper(it.Severity), it.ID, it.Title)
		if it.Category != "" {
			fmt.Fprintf(&b, " (Kategoria: %s)", it.Category)
		}
		b.WriteByte('\n')
		if it.URL != "" {
			fmt.Fprintf(&b, "   URL: %s\n", it.URL)
		}
		for _, ev := range it.Evidence {
			fmt.Fprintf(&b, "   Obserwacja: %s\n", ev)
		}
		if it.Rec != "" {
			fmt.Fprintf(&b, "   Zalecenie: %s\n", it.Rec)
		}
		b.WriteByte('\n')
	}

	b.WriteString("ZADANIE DLA MODELU LLM:\n")
	b.WriteString("1. Przygotuj kompletne, gotowe do skopiowania i wdrożenia fragmenty kodu (np. HTML w sekcji <head>, nagłówki odpowiedzi HTTP, plik robots.txt, sitemap.xml lub dane strukturalne JSON-LD Schema.org).\n")
	b.WriteString("2. Dla każdego błędu opisz krótko przyczynę powstania i instrukcję weryfikacji po wdrożeniu.\n")
	b.WriteString("3. Rozwiązania muszą spełniać standardy W3C oraz wytyczne Google Search Essentials / Search Central.")
	return b.String()
}
