package audit

import (
	"github.com/pan-dolina/crawlgrade/internal/crawler"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/report"
	"github.com/pan-dolina/crawlgrade/internal/terms"
	"sort"
	"strings"
)

func enrichReport(rep *report.Report, c *crawler.Result, keywords []string) {
	var zones []terms.Zones
	inbound := map[string]int{}
	outbound := map[string]int{}
	for _, p := range parsedPages(c) {
		seen := map[string]bool{}
		for _, l := range p.Page.Links {
			if l.URL != "" && !l.External && !l.NoFollow && l.URL != p.Page.URL && !seen[l.URL] {
				seen[l.URL] = true
				outbound[p.Page.URL]++
				if p.Page.Robots.Indexable() {
					inbound[l.URL]++
				}
			}
		}
		if !p.Page.Robots.Indexable() {
			continue
		}
		var headings []string
		for _, h := range p.Page.Headings {
			headings = append(headings, h.Text)
		}
		zones = append(zones, terms.Zones{URL: p.Page.URL, Title: p.Page.Title(), Description: p.Page.Description(), Headings: strings.Join(headings, " "), Body: p.Content.Text})
	}
	w := terms.Weigh(zones, keywords)
	rep.Terms = w.Site
	for _, p := range c.Pages {
		m := report.PageMetric{URL: p.URL, Depth: p.Depth}
		if p.Response != nil {
			m.Status = p.Response.Status
		}
		if pp, ok := p.Data.(parsedPage); ok {
			m.URL = pp.Page.URL
			m.Indexable = pp.Page.Robots.Indexable()
			m.Terms = w.Pages[m.URL]
		}
		m.Inbound, m.Outbound = inbound[m.URL], outbound[m.URL]
		rep.Pages = append(rep.Pages, m)
	}
	sort.Slice(rep.Pages, func(i, j int) bool { return rep.Pages[i].URL < rep.Pages[j].URL })
	penalty, hygiene := 0, 0
	weights := map[findings.Severity]int{findings.SeverityInfo: 0, findings.SeverityLow: 1, findings.SeverityMedium: 3, findings.SeverityHigh: 8, findings.SeverityCritical: 20}
	for group, fs := range rep.Groups {
		for _, f := range fs {
			if group == report.GroupWebHygiene {
				hygiene += weights[f.Severity]
			} else {
				penalty += weights[f.Severity]
			}
		}
	}
	if len(zones) == 0 {
		return
	}
	n := len(zones)
	rep.Scores = &report.Scores{SEO: max(0, 100-penalty/n), WebHygiene: max(0, 100-hygiene/n)}
}
