package htmlcheck

import "testing"

func TestLanguageValidation(t *testing.T) {
	for _, tc := range []struct {
		tag   string
		valid bool
	}{{"pl", true}, {"en-GB", true}, {"zh-Hant-TW", true}, {"x-default", true}, {"en-UK", false}, {"en_US", false}, {"zz", false}, {"english", false}, {"", false}} {
		if got := validHreflang(tc.tag); got != tc.valid {
			t.Errorf("%q: %v", tc.tag, got)
		}
	}
}

func TestHreflangReciprocityAndEmptyValue(t *testing.T) {
	a := parse(t, "https://example.com/", `<html lang="pl"><link rel="alternate" hreflang="pl" href="/"><link rel="alternate" hreflang="en" href="/en">`)
	b := parse(t, "https://example.com/en", `<html lang="en">`)
	if got := ids(HreflangSiteFindings([]*Page{a, b})); got != "SEO-HREFLANG-007" {
		t.Fatal(got)
	}
	c := parse(t, "https://example.com/", `<link rel="alternate" hreflang="" href="/">`)
	if got := ids(c.HreflangFindings()); got != "SEO-HREFLANG-003" {
		t.Fatal(got)
	}
}

func TestImageDecorativeAltAndSocialRequired(t *testing.T) {
	p := parse(t, "https://example.com/", `<img src="/a.png" alt="" width="20" height="20"><img src="/b.png"><meta property="og:title" content="Hello">`)
	if len(p.ImageFindings()) != 2 {
		t.Fatal(p.ImageFindings())
	}
	if len(p.SocialRequiredFindings()) != 1 {
		t.Fatal(p.SocialRequiredFindings())
	}
}
