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
html { scroll-behavior: smooth; }
body { font: 15px/1.5 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; margin: 0; background: #fafafa; color: #1a1a1a; }
main { max-width: 52rem; margin: 0 auto; padding: 2rem 1.25rem 4rem; }
#top { scroll-margin-top: 1rem; }
:target { scroll-margin-top: 1.25rem; }
h1 { font-size: 1.4rem; margin: 0 0 .25rem; word-break: break-word; }
.url { color: #555; word-break: break-all; margin: 0 0 1.5rem; }
.banner { display: flex; flex-wrap: wrap; gap: .5rem; margin-bottom: 1.5rem; }
.stat { border: 1px solid #ddd; border-radius: .5rem; padding: .5rem .75rem; background: #fff; }
.stat-link { color: inherit; text-decoration: none; display: block; cursor: pointer; transition: border-color .15s, box-shadow .15s; }
.stat-link:hover, .stat-link:focus-visible { border-color: #777; box-shadow: 0 0 0 2px #bbb; }
.stat b { display: block; font-size: 1.5rem; line-height: 1.1; }
.stat.critical b { color: #b00020; }
.stat.high b { color: #e65100; }
.stat.medium b { color: #b45309; }
.stat.low b { color: #2e7d32; }
.stat.info b { color: #546e7a; }
h2 { font-size: 1.1rem; border-bottom: 1px solid #ddd; padding-bottom: .25rem; margin-top: 2rem; }
summary h2, summary .section-title { display: inline; font-size: 1.1rem; border-bottom: none; margin: 0; padding: 0; color: inherit; }
table { border-collapse: collapse; width: 100%; margin: .5rem 0; }
th, td { text-align: left; padding: .4rem .5rem; border-bottom: 1px solid #eee; vertical-align: top; }
th { font-size: .8rem; text-transform: uppercase; letter-spacing: .03em; color: #666; }
.finding { border: 1px solid #eee; border-radius: .5rem; padding: .5rem .75rem; margin: .75rem 0; background: #fff; }
.finding .sev { font-weight: 700; margin-right: .4rem; }
.sev-critical { color: #b00020; }
.sev-high { color: #e65100; }
.sev-medium { color: #b45309; }
.sev-low { color: #2e7d32; }
.sev-info { color: #546e7a; }
.finding .title { font-weight: 600; }
.quickwins { display: grid; grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr)); gap: .75rem; margin-top: .5rem; }
.quickwin { border: 1px solid #cbd5e1; border-left: .35rem solid #e65100; border-radius: .5rem; padding: .85rem; background: #f8fafc; box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08); }
.quickwin.sev-border-critical { border-left-color: #b00020; }
.quickwin.sev-border-high { border-left-color: #c2410c; }
.quickwin.sev-border-medium { border-left-color: #b45309; }
.quickwin.sev-border-low { border-left-color: #15803d; }
.quickwin.sev-border-info { border-left-color: #0369a1; }
.quickwin .title { font-weight: 700; color: #0f172a; }
.quickwin .url-line { color: #334155; font-size: .85rem; word-break: break-all; margin: .25rem 0; }
.quickwin .evidence { color: #1e293b; margin: .35rem 0 0; padding-left: 1.2rem; }
.quickwin .rec { color: #0f172a; font-size: .9rem; font-weight: 500; margin-top: .45rem; padding-top: .45rem; border-top: 1px dashed #cbd5e1; }
details { border: 1px solid #ddd; border-radius: .5rem; margin: .65rem 0; background: #fff; }
details:target { border-color: #0056b3; box-shadow: 0 0 0 3px rgba(0, 86, 179, 0.25); }
summary { cursor: pointer; padding: .75rem; font-weight: 700; user-select: none; }
summary .count { color: #666; font-weight: 400; }
.detail-body { padding: 0 .75rem .75rem; }
.category { color: #666; font-size: .8rem; text-transform: uppercase; letter-spacing: .03em; }
.url-line { color: #555; font-size: .85rem; word-break: break-all; margin: .25rem 0; }
.evidence { margin: .35rem 0 0; padding-left: 1.2rem; color: #333; }
.rec { color: #444; font-size: .9rem; margin-top: .35rem; }
.empty { color: #666; font-style: italic; }
.llm-section { margin-top: 1rem; border-color: #818cf8; background: #faf5ff; }
.llm-section summary { color: #4338ca; }
.llm-intro { color: #475569; font-size: .9rem; margin: 0 0 .5rem; }
.llm-prompt { background: #0f172a; color: #f8fafc; padding: .85rem 1rem; border-radius: .375rem; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: .82rem; line-height: 1.45; white-space: pre-wrap; word-break: break-word; overflow-x: auto; border: 1px solid #1e293b; user-select: all; }
.llm-hint { margin-top: .6rem; border: 1px solid #cbd5e1; border-radius: .375rem; background: #f1f5f9; }
.llm-hint summary { padding: .4rem .6rem; font-size: .8rem; font-weight: 600; color: #475569; }
.llm-hint .detail-body { padding: 0 .6rem .6rem; }
.llm-hint .llm-prompt { background: #1e293b; padding: .5rem .75rem; font-size: .78rem; margin: 0; }
footer { margin-top: 2rem; color: #888; font-size: .8rem; }
.to-top { position: fixed; right: 1.25rem; bottom: 1.25rem; z-index: 100; border: 1px solid #bbb; border-radius: 999px; padding: .5rem .85rem; background: #fff; color: #333; text-decoration: none; font-size: .85rem; font-weight: 600; box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15); display: inline-flex; align-items: center; gap: .25rem; transition: background .15s, box-shadow .15s, transform .15s; }
.to-top:hover, .to-top:focus-visible { background: #f0f0f0; box-shadow: 0 4px 14px rgba(0, 0, 0, 0.2); transform: translateY(-2px); }
@media (prefers-color-scheme: dark) {
  body { background: #121212; color: #e6e6e6; }
  .stat, .finding, details { background: #1e1e1e; border-color: #333; }
  details:target { border-color: #4da3ff; box-shadow: 0 0 0 3px rgba(77, 163, 255, 0.3); }
  .stat-link:hover, .stat-link:focus-visible { border-color: #aaa; box-shadow: 0 0 0 2px #555; }
  .sev-critical { color: #f87171; }
  .sev-high { color: #fb923c; }
  .sev-medium { color: #fbbf24; }
  .sev-low { color: #4ade80; }
  .sev-info { color: #38bdf8; }
  .quickwin { background: #18202c; border-color: #334155; box-shadow: 0 2px 6px rgba(0, 0, 0, 0.4); }
  .quickwin .title { color: #f8fafc; }
  .quickwin .url-line { color: #94a3b8; }
  .quickwin .evidence { color: #cbd5e1; }
  .quickwin .rec { color: #f1f5f9; border-top-color: #334155; }
  .quickwin.sev-border-critical { border-left-color: #ef4444; }
  .quickwin.sev-border-high { border-left-color: #f97316; }
  .quickwin.sev-border-medium { border-left-color: #f59e0b; }
  .quickwin.sev-border-low { border-left-color: #22c55e; }
  .quickwin.sev-border-info { border-left-color: #38bdf8; }
  .llm-section { background: #151329; border-color: #6366f1; }
  .llm-section summary { color: #c7d2fe; }
  .llm-intro { color: #94a3b8; }
  .llm-prompt { background: #020617; border-color: #1e293b; color: #e2e8f0; }
  .llm-hint { background: #1e293b; border-color: #334155; }
  .llm-hint summary { color: #94a3b8; }
  .llm-hint .llm-prompt { background: #0f172a; }
  th, .url, .url-line, footer { color: #aaa; }
  th { border-color: #333; }
  td { border-color: #2a2a2a; }
  .evidence, .rec { color: #ccc; }
  .to-top { background: #222; color: #eee; border-color: #444; box-shadow: 0 2px 10px rgba(0, 0, 0, 0.5); }
  .to-top:hover, .to-top:focus-visible { background: #333; box-shadow: 0 4px 14px rgba(0, 0, 0, 0.7); }
}
</style>
</head>
<body>
<main id="top">
<h1>CrawlGrade report</h1>
<p class="url">{{.StartURL}}</p>
<div class="banner">
{{if and (not .QuickOnly) (gt .Counts.Critical 0)}}<a class="stat stat-link critical" href="#findings-critical"><b>{{.Counts.Critical}}</b>critical</a>{{else}}<div class="stat critical"><b>{{.Counts.Critical}}</b>critical</div>{{end}}
{{if and (not .QuickOnly) (gt .Counts.High 0)}}<a class="stat stat-link high" href="#findings-high"><b>{{.Counts.High}}</b>high</a>{{else}}<div class="stat high"><b>{{.Counts.High}}</b>high</div>{{end}}
{{if and (not .QuickOnly) (gt .Counts.Medium 0)}}<a class="stat stat-link medium" href="#findings-medium"><b>{{.Counts.Medium}}</b>medium</a>{{else}}<div class="stat medium"><b>{{.Counts.Medium}}</b>medium</div>{{end}}
{{if and (not .QuickOnly) (gt .Counts.Low 0)}}<a class="stat stat-link low" href="#findings-low"><b>{{.Counts.Low}}</b>low</a>{{else}}<div class="stat low"><b>{{.Counts.Low}}</b>low</div>{{end}}
{{if and (not .QuickOnly) (gt .Counts.Info 0)}}<a class="stat stat-link info" href="#findings-info"><b>{{.Counts.Info}}</b>info</a>{{else}}<div class="stat info"><b>{{.Counts.Info}}</b>info</div>{{end}}
{{if and (not .QuickOnly) .Pages}}<a class="stat stat-link" href="#pages"><b>{{.Pages}}</b>pages</a>{{else}}<div class="stat"><b>{{.Pages}}</b>pages</div>{{end}}
</div>
<details id="terms" open>
<summary><h2 class="section-title">Site terms</h2>{{if .Report.Terms}} <span class="count">({{len .Report.Terms}})</span>{{end}}</summary>
<div class="detail-body">
{{if .Report.Terms}}<table><tr><th>Term</th><th>Strength</th></tr>{{range .Report.Terms}}<tr><td>{{.Term}}</td><td>{{.Strength}}</td></tr>{{end}}</table>{{else}}<p class="empty">No term data available.</p>{{end}}
</div>
</details>
<details id="quick-wins" open>
<summary><h2 class="section-title">Quick wins</h2>{{if .QuickWins}} <span class="count">({{len .QuickWins}})</span>{{end}}</summary>
<div class="detail-body">
{{if .QuickWins}}
<div class="quickwins">
{{range .QuickWins}}
<div class="quickwin sev-border-{{.Severity}}">
<div><span class="sev sev-{{.Severity}}">{{.Severity}}</span> <span class="title">{{.ID}} {{.Title}}</span></div>
{{if .URL}}<p class="url-line">{{.URL}}</p>{{end}}
{{if .Evidence}}<p class="evidence">{{index .Evidence 0}}</p>{{end}}
<p class="rec">{{.Rec}}</p>
{{if .LLMInstruction}}
<details class="llm-hint">
<summary>LLM instructions</summary>
<div class="detail-body"><pre class="llm-prompt"><code>{{.LLMInstruction}}</code></pre></div>
</details>
{{end}}
</div>
{{end}}
</div>
{{else}}
<p class="empty">No high-priority actions found.</p>
{{end}}
</div>
</details>
{{if .LLMPrompt}}
<details id="llm-instructions" class="llm-section">
<summary><h2 class="section-title">LLM remediation prompt</h2></summary>
<div class="detail-body">
<p class="llm-intro">Copy and paste the prompt below into an AI model (e.g. ChatGPT, Claude, Cursor) to generate fixes and code for the detected issues:</p>
<pre class="llm-prompt"><code>{{.LLMPrompt}}</code></pre>
</div>
</details>
{{end}}
{{if .QuickOnly}}<p class="empty">Focused view. Use the regular HTML format for the complete finding list.</p>{{else}}
{{if .Severities}}<h2>Findings by priority</h2>
{{range .Severities}}<details id="findings-{{.Severity}}" open><summary><span class="sev sev-{{.Severity}}">{{.Title}}</span> <span class="count">({{.Count}})</span></summary><div class="detail-body">{{range .Findings}}
<div class="finding">
<div><span class="category">{{.Category}}</span> <span class="title">{{.ID}} {{.Title}}</span></div>
{{if .URL}}<p class="url-line">{{.URL}}</p>{{end}}
{{if .Evidence}}<ul class="evidence">{{range .Evidence}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{if .Rec}}<p class="rec">{{.Rec}}</p>{{end}}
{{if .LLMInstruction}}
<details class="llm-hint">
<summary>LLM instructions</summary>
<div class="detail-body"><pre class="llm-prompt"><code>{{.LLMInstruction}}</code></pre></div>
</details>
{{end}}
</div>
{{end}}</div></details>{{end}}
{{else}}
<p class="empty">No findings.</p>
{{end}}
{{if .Report.Scores}}<p>Technical SEO score: {{.Report.Scores.SEO}}/100. Passive web hygiene: {{.Report.Scores.WebHygiene}}/100. Scores describe observed checks, not rankings.</p>{{end}}
{{if .Report.Baseline}}<h2>Baseline comparison</h2><p>{{len .Report.Baseline.New}} new findings; {{len .Report.Baseline.Resolved}} resolved. Page delta: {{.Report.Baseline.PageDelta}}. SEO score delta: {{.Report.Baseline.SEOScoreDelta}}.</p>{{end}}
{{if .Report.Pages}}<details id="pages"><summary><h2 class="section-title">Pages and internal links</h2> <span class="count">({{len .Report.Pages}})</span></summary><div class="detail-body"><table><tr><th>URL</th><th>Status</th><th>Inbound</th><th>Outbound</th><th>Terms</th></tr>{{range .Report.Pages}}<tr><td>{{.URL}}</td><td>{{.Status}}</td><td>{{.Inbound}}</td><td>{{.Outbound}}</td><td>{{range .Terms}}{{.Term}} ({{.Strength}}); {{end}}</td></tr>{{end}}</table></div></details>{{end}}
{{if .Stop}}<p>Crawl stopped: <strong>{{.Stop}}</strong></p>{{end}}
{{end}}
<footer>Generated by CrawlGrade. No data leaves this document.</footer>
</main>
<a class="to-top" href="#top" aria-label="Back to top">↑ Back to top</a>
</body>
</html>
`

// templateSet builds the parsed template. The layout is a constant, so this
// only parses once per process.
func loadHTMLTemplate() (*template.Template, error) {
	return template.New("report").Parse(htmlTemplateContent)
}
