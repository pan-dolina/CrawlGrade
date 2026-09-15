package urlnorm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return u
}

func TestNormalizeEdgeCases(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.com", "https://example.com/"},
		{"https://EXAMPLE.com/Path/To/Page", "https://example.com/Path/To/Page"},
		{"https://example.com/a/", "https://example.com/a/"},
		{"https://example.com/a?", "https://example.com/a"},
		{"https://example.com/%61%62%63", "https://example.com/abc"},
		{"https://example.com/%e2%82%ac", "https://example.com/%E2%82%AC"},
		{"https://example.com/€", "https://example.com/%E2%82%AC"},
		{"https://example.com/a b", "https://example.com/a%20b"},
		{"https://example.com/100%", "https://example.com/100%25"},
		{"https://example.com/%zz", "https://example.com/%25zz"},
		{"https://example.com/a/%2e%2e/b", "https://example.com/b"},
		{"https://example.com/../../x", "https://example.com/x"},
		{"https://example.com/a/b/..", "https://example.com/a/"},
		{"https://example.com/a/.", "https://example.com/a/"},
		{"https://example.com/?q=a+b&x=%7e", "https://example.com/?q=a+b&x=~"},
		{"https://example.com/?q=zażółć", "https://example.com/?q=za%C5%BC%C3%B3%C5%82%C4%87"},
		{"https://example.com:0443/", "https://example.com/"},
		{"https://example.com:08080/", "https://example.com:8080/"},
		{"http://[::1]:80/", "http://[::1]/"},
		{"http://[2001:DB8::1]/", "http://[2001:db8::1]/"},
		{"http://192.168.000.1/", ""}, // leading zeros are not an IP literal here
		{"https://ŁÓDŹ.example/", "https://xn--d-uga0v4h.example/"},
		{"https://faß.de/", "https://xn--fa-hia.de/"},
		{"https://_dmarc.example.com/", "https://_dmarc.example.com/"},
		{"https://example.com/#", "https://example.com/"},
	}
	for _, c := range cases {
		u, err := Parse(c.in)
		if c.want == "" {
			if err == nil && strings.Contains(u.Host, "000") {
				// Accepted as a host name; the network policy classifies what
				// it resolves to. Nothing more to check.
				continue
			}
			if err != nil {
				continue
			}
		}
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if u.String() != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, u, c.want)
		}
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	inputs := []string{
		"https://example.com/a%2fb/%7e?x=%41&y=1 2",
		"https://bücher.example/ü/?ä=ö",
		"http://[::ffff:1.2.3.4]:8080/./a/../b",
		"https://example.com/%25/%",
	}
	for _, in := range inputs {
		u := mustParse(t, in)
		again := mustParse(t, u.String())
		if again.String() != u.String() {
			t.Errorf("not idempotent: %q -> %q -> %q", in, u, again)
		}
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]error{
		"javascript:alert(1)":               ErrNotHTTP,
		"mailto:a@example.com":              ErrNotHTTP,
		"tel:+48123456789":                  ErrNotHTTP,
		"data:text/html,<script>":           ErrNotHTTP,
		"ftp://example.com/":                ErrNotHTTP,
		"https://user:pass@example.com/":    ErrCredentials,
		"https://user@example.com/":         ErrCredentials,
		"/relative/path":                    ErrInvalidURL,
		"https:///nohost":                   ErrInvalidURL,
		"https://exa%6Dple.com/":            ErrInvalidURL,
		"https://example.com:abc/":          ErrInvalidURL,
		"https://example.com:0/":            ErrInvalidURL,
		"https://example.com:123456/":       ErrInvalidURL,
		"https://exa mple.com/":             ErrInvalidURL,
		"https://example.com/" + long(3000): ErrTooLong,
		"https://" + long(300) + ".com/":    ErrInvalidURL,
		"https://a\u202eb.example/":         ErrInvalidURL, // BiDi override in host
		"https://xn--a.example/":            ErrInvalidURL,
	}
	for in, want := range cases {
		if _, err := Parse(in); !errors.Is(err, want) {
			t.Errorf("Parse(%.60q) error = %v, want %v", in, err, want)
		}
	}
	if _, err := ParseStart("   "); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("empty start: %v", err)
	}
}

func long(n int) string { return strings.Repeat("a", n) }

func TestResolveLikeABrowser(t *testing.T) {
	base := mustParse(t, "https://example.com/dir/page.html?x=1")
	cases := map[string]string{
		"other.html":                "https://example.com/dir/other.html",
		"../up":                     "https://example.com/up",
		"/abs":                      "https://example.com/abs",
		"//cdn.example.com/lib.js":  "https://cdn.example.com/lib.js",
		"?y=2":                      "https://example.com/dir/page.html?y=2",
		"#section":                  "https://example.com/dir/page.html?x=1",
		"":                          "https://example.com/dir/page.html?x=1",
		"  /padded  ":               "https://example.com/padded",
		"/new\nline\ttab":           "https://example.com/newlinetab",
		"\x00\x1f/controls":         "https://example.com/controls",
		"HTTP://EXAMPLE.COM:443/":   "http://example.com:443/",
		"https://example.com:443/x": "https://example.com/x",
	}
	for ref, want := range cases {
		u, err := Resolve(base, ref)
		if err != nil {
			t.Errorf("Resolve(%q): %v", ref, err)
			continue
		}
		if u.String() != want {
			t.Errorf("Resolve(%q) = %q, want %q", ref, u, want)
		}
	}
	if _, err := Resolve(base, " javascript:void(0)"); !errors.Is(err, ErrNotHTTP) {
		t.Errorf("javascript href: %v", err)
	}
	if _, err := Resolve(base, "http://[::1"); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("broken IPv6 href: %v", err)
	}
}

func TestScope(t *testing.T) {
	s := NewScope(mustParse(t, "https://www.example.com/start"))
	if s.Key() != "example.com" {
		t.Fatalf("key = %q", s.Key())
	}
	in := []string{
		"https://example.com/", "http://example.com/a", "https://www.example.com/b",
		"https://WWW.EXAMPLE.COM/c", "http://example.com:80/", "https://example.com:443/",
	}
	out := []string{
		"https://sub.example.com/", "https://example.com.evil.test/", "https://example.com:8443/",
		"https://wwwexample.com/", "https://www.www.example.com/", "https://notexample.com/",
	}
	for _, s2 := range in {
		if !s.Contains(mustParse(t, s2)) {
			t.Errorf("%s should be in scope", s2)
		}
	}
	for _, s2 := range out {
		if s.Contains(mustParse(t, s2)) {
			t.Errorf("%s should be out of scope", s2)
		}
	}
	local := NewScope(mustParse(t, "http://127.0.0.1:8080/"))
	if !local.Contains(mustParse(t, "http://127.0.0.1:8080/x")) || local.Contains(mustParse(t, "http://127.0.0.1:9090/x")) {
		t.Error("ports must be part of the site key")
	}
	if s.Contains(nil) || s.Contains(&url.URL{Scheme: "ftp", Host: "example.com"}) {
		t.Error("nil or non-http URL in scope")
	}
}

func TestTrapGuard(t *testing.T) {
	g := NewTrapGuard(TrapLimits{})
	admit := func(s string) TrapReason { return g.Admit(mustParse(t, s)) }

	if r := admit("https://example.com/a/b/a/b/a/b/a/"); r != TrapRepeatedSegment {
		t.Errorf("repeated segments: %q", r)
	}
	if r := admit("https://example.com/a/b/a/b/a/b/"); r != TrapNone {
		t.Errorf("three repeats should be allowed: %q", r)
	}
	if r := admit("https://example.com/" + strings.Repeat("x/", 31)); r != TrapTooManySegments && r != TrapRepeatedSegment {
		t.Errorf("deep path: %q", r)
	}
	var segs []string
	for i := range 31 {
		segs = append(segs, fmt.Sprintf("s%c", 'a'+rune(i%26))+strings.Repeat("z", i))
	}
	if r := admit("https://example.com/" + strings.Join(segs, "/")); r != TrapTooManySegments {
		t.Errorf("31 distinct segments: %q", r)
	}

	// Query explosion: 25 variants per path.
	for i := range 30 {
		r := admit(fmt.Sprintf("https://example.com/search?q=%d&sort=x", i))
		if (i < 25) != (r == TrapNone) {
			t.Errorf("query variant %d: %q", i, r)
		}
	}
	// Admitting an already admitted URL is free.
	if r := admit("https://example.com/search?q=0&sort=x"); r != TrapNone {
		t.Errorf("re-admission: %q", r)
	}

	// Infinite calendar: /calendar/2024/01/01 ... share one pattern.
	rejected := 0
	for d := range 150 {
		if admit(fmt.Sprintf("https://example.com/calendar/%d/%02d/", 2000+d/12, d%12+1)) == TrapPatternExplosion {
			rejected++
		}
	}
	if rejected != 50 {
		t.Errorf("calendar URLs rejected = %d, want 50", rejected)
	}

	g2 := NewTrapGuard(TrapLimits{MaxURLLength: 30})
	if r := g2.Admit(mustParse(t, "https://example.com/a-rather-long-path")); r != TrapURLTooLong {
		t.Errorf("long URL: %q", r)
	}
}

func TestPattern(t *testing.T) {
	cases := map[string]string{
		"https://example.com/2024/01/31/post-7":    "/{n}/{n}/{n}/post-{n}",
		"https://example.com/list?page=2&sort=asc": "/list?page&sort",
		"https://example.com/list?sort=asc&page=2": "/list?page&sort",
		"https://example.com/p?id1=5&id2=6&flag":   "/p?flag&id{n}&id{n}",
		"https://example.com/":                     "/",
	}
	for in, want := range cases {
		if got := Pattern(mustParse(t, in)); got != want {
			t.Errorf("Pattern(%s) = %q, want %q", in, got, want)
		}
	}
}
