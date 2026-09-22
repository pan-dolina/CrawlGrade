package report

import "html/template"

// htmlTemplate is the standalone HTML report layout. It is embedded as a
// string so the binary needs no external assets. All values are rendered
// through html/template, which escapes them per context; there is no script
// and no network dependency.
const htmlTemplateContent = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CrawlGrade report — {{.StartURL}}</title>
<style>
:root { color-scheme: light dark; }
body { font: 15px/1.5 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; margin: 0; background: #fafafa; color: #1a1a1a; }
main { max-width: 52rem; margin: 0 auto; padding: 2rem 1.25rem 4rem; }
h1 { font-size: 1.4rem; margin: 0 0 .25rem; word-break: break-word; }
.url { color: #555; word-break: break-all; margin: 0 0 1.5rem; }
.banner { display: flex; flex-wrap: wrap; gap: .5rem; margin-bottom: 1.5rem; }
.stat { border: 1px solid #ddd; border-radius: .5rem; padding: .5rem .75rem; background: #fff; }
.stat b { display: block; font-size: 1.5rem; line-height: 1.1; }
.stat.critical b { color: #b00020; }
.stat.high b { color: #e65100; }
.stat.medium b { color: #f9a825; }
.stat.low b { color: #2e7d32; }
.stat.info b { color: #546e7a; }
h2 { font-size: 1.1rem; border-bottom: 1px solid #ddd; padding-bottom: .25rem; margin-top: 2rem; }
table { border-collapse: collapse; width: 100%; margin: .5rem 0; }
th, td { text-align: left; padding: .4rem .5rem; border-bottom: 1px solid #eee; vertical-align: top; }
th { font-size: .8rem; text-transform: uppercase; letter-spacing: .03em; color: #666; }
.finding { border: 1px solid #eee; border-radius: .5rem; padding: .5rem .75rem; margin: .75rem 0; background: #fff; }
.finding .sev { font-weight: 700; margin-right: .4rem; }
.sev-critical { color: #b00020; }
.sev-high { color: #e65100; }
.sev-medium { color: #f9a825; }
.sev-low { color: #2e7d32; }
.sev-info { color: #546e7a; }
.finding .title { font-weight: 600; }
.quickwins { display: grid; grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr)); gap: .75rem; }
.quickwin { border: 1px solid #f0c36d; border-left: .35rem solid #e65100; border-radius: .5rem; padding: .75rem; background: #fffaf0; }
.quickwin .title { font-weight: 700; }
.url-line { color: #555; font-size: .85rem; word-break: break-all; margin: .25rem 0; }
.evidence { margin: .35rem 0 0; padding-left: 1.2rem; color: #333; }
.rec { color: #444; font-size: .9rem; margin-top: .35rem; }
.empty { color: #666; font-style: italic; }
footer { margin-top: 2rem; color: #888; font-size: .8rem; }
@media (prefers-color-scheme: dark) {
  body { background: #121212; color: #e6e6e6; }
  .stat, .finding { background: #1e1e1e; border-color: #333; }
  th, .url, .url-line, footer { color: #aaa; }
  th { border-color: #333; }
  td { border-color: #2a2a2a; }
  .evidence, .rec { color: #ccc; }
}
</style>
</head>
<body>
<main>
<h1>CrawlGrade report</h1>
<p class="url">{{.StartURL}}</p>
<div class="banner">
<div class="stat critical"><b>{{.Counts.Critical}}</b>critical</div>
<div class="stat high"><b>{{.Counts.High}}</b>high</div>
<div class="stat medium"><b>{{.Counts.Medium}}</b>medium</div>
<div class="stat low"><b>{{.Counts.Low}}</b>low</div>
<div class="stat info"><b>{{.Counts.Info}}</b>info</div>
<div class="stat"><b>{{.Pages}}</b>pages</div>
</div>
{{if .Report.Scores}}<p>Technical SEO score: {{.Report.Scores.SEO}}/100. Passive web hygiene: {{.Report.Scores.WebHygiene}}/100. Scores describe observed checks, not rankings.</p>{{end}}
<h2>Quick wins</h2>
{{if .QuickWins}}<div class="quickwins">{{range .QuickWins}}<div class="quickwin"><div><span class="sev sev-{{.Severity}}">{{.Severity}}</span> <span class="title">{{.ID}} {{.Title}}</span></div>{{if .URL}}<p class="url-line">{{.URL}}</p>{{end}}{{if .Evidence}}<p class="evidence">{{index .Evidence 0}}</p>{{end}}<p class="rec">{{.Rec}}</p></div>{{end}}</div>{{else}}<p class="empty">No high-priority actions found.</p>{{end}}
{{if .QuickOnly}}<p class="empty">Focused view. Use the regular HTML format for the complete finding list.</p>{{else}}
{{if .Report.Baseline}}<h2>Baseline comparison</h2><p>{{len .Report.Baseline.New}} new findings; {{len .Report.Baseline.Resolved}} resolved. Page delta: {{.Report.Baseline.PageDelta}}. SEO score delta: {{.Report.Baseline.SEOScoreDelta}}.</p>{{end}}
{{if .Report.Terms}}<h2>Site terms</h2><table><tr><th>Term</th><th>Strength</th></tr>{{range .Report.Terms}}<tr><td>{{.Term}}</td><td>{{.Strength}}</td></tr>{{end}}</table>{{end}}
{{if .Report.Pages}}<h2>Pages and internal links</h2><table><tr><th>URL</th><th>Status</th><th>Inbound</th><th>Outbound</th><th>Terms</th></tr>{{range .Report.Pages}}<tr><td>{{.URL}}</td><td>{{.Status}}</td><td>{{.Inbound}}</td><td>{{.Outbound}}</td><td>{{range .Terms}}{{.Term}} ({{.Strength}}); {{end}}</td></tr>{{end}}</table>{{end}}
{{if .Stop}}<p>Crawl stopped: <strong>{{.Stop}}</strong></p>{{end}}
{{if .Groups}}
{{range .Groups}}
<h2>{{.Title}} <span class="empty">{{len .Findings}} finding{{if ne (len .Findings) 1}}s{{end}}</span></h2>
{{range .Findings}}
<div class="finding">
<div><span class="sev sev-{{.Severity}}">{{.Severity}}</span><span class="title">{{.ID}} {{.Title}}</span></div>
{{if .URL}}<p class="url-line">{{.URL}}</p>{{end}}
{{if .Evidence}}<ul class="evidence">{{range .Evidence}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{if .Rec}}<p class="rec">{{.Rec}}</p>{{end}}
</div>
{{end}}
{{end}}
{{else}}
<p class="empty">No findings.</p>
{{end}}
{{end}}
<footer>Generated by CrawlGrade. No data leaves this document.</footer>
</main>
</body>
</html>
`

// templateSet builds the parsed template. The layout is a constant, so this
// only parses once per process.
func loadHTMLTemplate() (*template.Template, error) {
	return template.New("report").Parse(htmlTemplateContent)
}
