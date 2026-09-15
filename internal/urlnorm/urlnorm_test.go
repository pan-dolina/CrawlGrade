package urlnorm

import (
	"net/url"
	"testing"
)

func TestNormalizeBasics(t *testing.T) {
	cases := map[string]string{
		"HTTP://Example.COM":             "http://example.com/",
		"https://example.com:443/a#frag": "https://example.com/a",
		"http://example.com:80/":         "http://example.com/",
		"http://example.com:8080/x":      "http://example.com:8080/x",
		"https://example.com/a/./b/../c": "https://example.com/a/c",
		"https://example.com/%7euser":    "https://example.com/~user",
		"https://example.com/a%2fb":      "https://example.com/a%2Fb",
		"https://example.com/?b=2&a=1":   "https://example.com/?b=2&a=1",
		"https://bücher.example/":        "https://xn--bcher-kva.example/",
		"https://example.com./":          "https://example.com/",
	}
	for in, want := range cases {
		u, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if u.String() != want {
			t.Errorf("Parse(%q) = %q, want %q", in, u, want)
		}
	}
}

func TestScopeAndResolve(t *testing.T) {
	start, _ := ParseStart("www.example.com")
	if start.String() != "https://www.example.com/" {
		t.Fatalf("start = %s", start)
	}
	s := NewScope(start)
	in, _ := Resolve(start, "/about")
	out, _ := Resolve(start, "https://other.example/")
	if !s.Contains(in) || s.Contains(out) {
		t.Error("scope")
	}
}

func TestTrapGuardBasics(t *testing.T) {
	g := NewTrapGuard(TrapLimits{MaxQueryVariants: 2})
	for i, want := range []TrapReason{TrapNone, TrapNone, TrapQueryExplosion} {
		u, _ := url.Parse("https://example.com/list?page=" + string(rune('1'+i)))
		if got := g.Admit(u); got != want {
			t.Errorf("variant %d: %q, want %q", i, got, want)
		}
	}
}
