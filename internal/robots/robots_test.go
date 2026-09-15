package robots

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
)

func TestRFC9309Matching(t *testing.T) {
	r := Parse([]byte(`User-Agent: *
Disallow: *.gif$
Disallow: /example/
Allow: /publications/

User-Agent: foobot
Disallow:/
Allow:/example/page.html
Allow:/example/allowed.gif

User-Agent: barbot
User-Agent: bazbot
Disallow: /example/page.html

User-Agent: quxbot
`))
	cases := []struct {
		agent, path string
		want        bool
	}{
		{"foobot", "/", false},
		{"foobot", "/example/page.html", true},
		{"foobot", "/example/allowed.gif", true},
		{"foobot", "/other", false},
		{"barbot", "/example/page.html", false},
		{"bazbot", "/example/page.html", false},
		{"barbot", "/example/other.html", true},
		{"quxbot", "/example/", true}, // empty group allows everything
		{"otherbot", "/example/", false},
		{"otherbot", "/image.gif", false},
		{"otherbot", "/image.gif?x", true},
		{"otherbot", "/publications/", true},
		{"FooBot", "/", false},
		{"otherbot", "/robots.txt", true},
	}
	for _, c := range cases {
		if got := r.Allowed(c.agent, c.path); got != c.want {
			t.Errorf("Allowed(%q, %q) = %v, want %v", c.agent, c.path, got, c.want)
		}
	}
}

func TestLongestMatchAndAllowTieBreak(t *testing.T) {
	r := Parse([]byte("user-agent: *\nallow: /folder\ndisallow: /folder\ndisallow: /folder/deeper\nallow: /folder/deeper/ok\n"))
	for path, want := range map[string]bool{
		"/folder":            true, // equal length: allow wins
		"/folder/deeper/x":   false,
		"/folder/deeper/ok1": true,
		"/other":             true,
	} {
		if got := r.Allowed(ProductToken, path); got != want {
			t.Errorf("%s: got %v", path, got)
		}
	}
	ok, rule := r.Match(ProductToken, "/folder/deeper/x")
	if ok || rule == nil || rule.Line != 4 {
		t.Errorf("deciding rule = %+v", rule)
	}
}

func TestWildcards(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"/", "/anything", true},
		{"/*.php$", "/index.php", true},
		{"/*.php$", "/index.php?x=1", false},
		{"/*.php", "/index.php?x=1", true},
		{"/fish*", "/fishheads/yummy.html", true},
		{"/fish*", "/Fish.asp", false},
		{"/*/private/*", "/a/private/b", true},
		{"/*/private/*", "/private/b", false},
		{"*", "/", true},
		{"/a*b*c$", "/axxbyycxc", true},
		{"/a*b*c$", "/axxbyycxd", false},
		{"$", "/", false},
		{"/***a", "/bbba", true},
		{"/%7Euser", "/~user", true},
		{"/caf%c3%a9", "/café", true},
		{"/caf%C3%A9", "/caf%c3%a9", true},
	}
	for _, c := range cases {
		if got := match(normalizePattern(c.pattern), normalizePattern(c.path)); got != c.want {
			t.Errorf("match(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestPathologicalPatternIsFast(t *testing.T) {
	pattern := strings.Repeat("*a", 1000) + "b$"
	target := "/" + strings.Repeat("a", 2047)
	start := time.Now()
	if match(pattern, target) {
		t.Error("unexpected match")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("match took %v", d)
	}
}

func TestParseToleratesMessyFiles(t *testing.T) {
	data := "\xEF\xBB\xBFDisallow: /before-group\r\n" +
		"User-agent: CrawlGrade/1.0 # our bot\r" +
		"Crawl-delay: 2.5\n" +
		"Disallow: /secret\n" +
		"Noindex: /x\n" +
		"garbage line\n" +
		"Sitemap: https://example.com/sitemap.xml\n" +
		"Sitemap: /relative.xml\n" +
		"user-agent: other\n" +
		"crawl-delay: soon\n" +
		"disallow: /" + strings.Repeat("x", MaxPatternLen+1) + "\n" +
		"\xff\xfe invalid utf8: yes\n" +
		strings.Repeat("y", maxLineLength+1) + "\n"
	r := Parse([]byte(data))
	if r.Allowed("crawlgrade", "/secret/page") || !r.Allowed("crawlgrade", "/before-group") {
		t.Error("group rules applied incorrectly")
	}
	if d, ok := r.CrawlDelay("crawlgrade"); !ok || d != 2500*time.Millisecond {
		t.Errorf("crawl delay = %v %v", d, ok)
	}
	if !r.HasGroupFor("crawlgrade") || r.HasGroupFor("nobody") {
		t.Error("HasGroupFor")
	}
	if len(r.Sitemaps) != 2 || r.Sitemaps[0].Line != 7 {
		t.Errorf("sitemaps = %+v", r.Sitemaps)
	}
	msgs := ""
	for _, w := range r.Warnings {
		msgs += w.Message + "\n"
	}
	for _, want := range []string{"outside of a user-agent group", `unknown directive "noindex"`, "no ':' separator", "invalid crawl-delay", "pattern too long", "line too long"} {
		if !strings.Contains(msgs, want) {
			t.Errorf("missing warning %q in:\n%s", want, msgs)
		}
	}
}

func TestParseLimits(t *testing.T) {
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	for b.Len() < MaxParseBytes+100 {
		b.WriteString("Disallow: /p\n")
	}
	r := Parse([]byte(b.String()))
	if !r.Truncated || r.RuleCount != MaxRules {
		t.Errorf("truncated=%v rules=%d", r.Truncated, r.RuleCount)
	}
	var w strings.Builder
	for range maxWarnings + 10 {
		w.WriteString("junk\n")
	}
	if got := len(Parse([]byte(w.String())).Warnings); got != maxWarnings {
		t.Errorf("warnings = %d", got)
	}
}

func TestUserAgentGroupsMerge(t *testing.T) {
	r := Parse([]byte("user-agent: crawlgrade\ndisallow: /a\n\nuser-agent: *\ndisallow: /b\n\nuser-agent: crawlgrade\ndisallow: /c\n"))
	if r.Allowed("crawlgrade", "/a") || r.Allowed("crawlgrade", "/c") || !r.Allowed("crawlgrade", "/b") {
		t.Error("specific groups must be merged and replace *")
	}
	if !r.Allowed("*", "/a") || r.Allowed("*", "/b") {
		t.Error("* lookup must use only * groups")
	}
}

type fakeFetcher struct {
	resp *fetcher.Response
	err  error
}

func (f fakeFetcher) Fetch(context.Context, fetcher.Request) (*fetcher.Response, error) {
	return f.resp, f.err
}

func ids(fs []findings.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.ID)
	}
	return strings.Join(s, ",")
}

func TestFetchOutcomes(t *testing.T) {
	site, _ := url.Parse("https://example.com/start")
	page, _ := url.Parse("https://example.com/page")
	robotsURL, _ := url.Parse("https://example.com/robots.txt")
	cases := []struct {
		name     string
		f        fakeFetcher
		outcome  Outcome
		allowed  bool
		findings string
	}{
		{"404", fakeFetcher{resp: &fetcher.Response{Status: 404}}, OutcomeNotFound, true, "SEO-ROBOTS-001"},
		{"403", fakeFetcher{resp: &fetcher.Response{Status: 403}}, OutcomeNotFound, true, "SEO-ROBOTS-001"},
		{"503", fakeFetcher{resp: &fetcher.Response{Status: 503}}, OutcomeUnavailable, false, "SEO-ROBOTS-002"},
		{"network", fakeFetcher{err: errors.New("connection refused")}, OutcomeUnavailable, false, "SEO-ROBOTS-002"},
		{"loop", fakeFetcher{resp: &fetcher.Response{Status: 302}, err: fetcher.ErrRedirectLoop}, OutcomeUnavailable, false, "SEO-ROBOTS-002"},
		{"no location", fakeFetcher{resp: &fetcher.Response{Status: 301}}, OutcomeUnavailable, false, "SEO-ROBOTS-002"},
		{"blocked", fakeFetcher{err: &netguard.BlockedError{Class: netguard.ClassMetadata}}, OutcomeBlocked, false, "SEO-ROBOTS-002"},
		{"ok", fakeFetcher{resp: &fetcher.Response{Status: 200, FinalURL: "https://example.com/robots.txt", Body: []byte("User-agent: *\nDisallow: /page\nSitemap: https://example.com/s.xml\n")}}, OutcomeOK, false, ""},
		{"ok no sitemap", fakeFetcher{resp: &fetcher.Response{Status: 200, Body: []byte("User-agent: *\nAllow: /\n")}}, OutcomeOK, true, "SEO-ROBOTS-006"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := Fetch(context.Background(), c.f, site)
			if file.Outcome != c.outcome {
				t.Errorf("outcome = %s", file.Outcome)
			}
			if got := file.Allowed(page); got != c.allowed {
				t.Errorf("allowed = %v", got)
			}
			if !file.Allowed(robotsURL) {
				t.Error("robots.txt itself must always be allowed")
			}
			if got := ids(file.Findings(site)); got != c.findings {
				t.Errorf("findings = %s, want %s", got, c.findings)
			}
		})
	}
}

func TestFindingsForProblematicFile(t *testing.T) {
	site, _ := url.Parse("https://example.com/")
	body := "User-agent: *\nDisallow: /\n\nUser-agent: crawlgrade\nAllow: /\nSitemap: sitemap.xml\nHost: example.com\n" + strings.Repeat("#", MaxParseBytes)
	file := Fetch(context.Background(), fakeFetcher{resp: &fetcher.Response{Status: 200, FinalURL: "https://www.example.com/robots.txt", Body: []byte(body)}}, site)
	got := ids(file.Findings(site))
	want := "SEO-ROBOTS-003,SEO-ROBOTS-005,SEO-ROBOTS-007,SEO-ROBOTS-008,SEO-ROBOTS-009"
	if got != want {
		t.Errorf("findings = %s, want %s", got, want)
	}
	if !file.Allowed(site) {
		t.Error("crawlgrade group allows everything")
	}

	blocksUs := Fetch(context.Background(), fakeFetcher{resp: &fetcher.Response{Status: 200, Body: []byte("User-agent: crawlgrade\nDisallow: /\nSitemap: https://example.com/s.xml")}}, site)
	if got := ids(blocksUs.Findings(site)); got != "SEO-ROBOTS-004" {
		t.Errorf("findings = %s", got)
	}
}

func TestFetchOverHTTP(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		if r.URL.Path != "/robots.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("User-agent: *\nCrawl-delay: 1\nUser-agent: crawlgrade\nCrawl-delay: 3\nDisallow: /tmp/\n"))
	}))
	defer srv.Close()
	site, _ := url.Parse(srv.URL + "/some/page?x=1")
	file := Fetch(context.Background(), fetcher.New(fetcher.Options{}), site)
	if file.Outcome != OutcomeOK || file.URL != srv.URL+"/robots.txt" || file.RuleCount != 1 || file.CrawlDelay != 3 {
		t.Fatalf("file = %+v", file)
	}
	if !strings.HasPrefix(accept, "text/plain") {
		t.Errorf("Accept = %q", accept)
	}
	tmp, _ := url.Parse(srv.URL + "/tmp/x")
	if file.Allowed(tmp) {
		t.Error("/tmp/ should be disallowed")
	}
}
