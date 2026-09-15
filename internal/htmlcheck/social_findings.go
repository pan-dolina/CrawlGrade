package htmlcheck

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// validHreflang reports whether value is a valid hreflang target: an ISO
// 639-1 language code, optionally followed by a hyphen and an ISO 3166-1
// alpha-2 region code. "x-default" is the special fallback marker.
func validHreflang(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "x-default" {
		return true
	}
	parts := strings.SplitN(value, "-", 2)
	if len(parts[0]) != 2 || !isAlpha2(parts[0]) {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	return len(parts[1]) == 2 && isAlpha2(parts[1])
}

func isAlpha2(s string) bool {
	if len(s) != 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

// HreflangFindings reports problems with the hreflang set of one page. A page
// with no hreflang links returns no findings; the "no hreflang" finding is
// produced by the caller when the page has language or regional versions.
func (p *Page) HreflangFindings() []findings.Finding {
	if len(p.Hreflangs) == 0 {
		return nil
	}
	var out []findings.Finding
	var invalid []string
	hasSelf := false
	seen := map[string]bool{}
	for _, h := range p.Hreflangs {
		if !validHreflang(h.Target) {
			invalid = append(invalid, "hreflang: "+findings.Truncate(h.Target, 60))
		}
		if h.Href == p.URL {
			hasSelf = true
		}
		seen[h.Href] = true
	}
	if len(invalid) > 0 {
		out = append(out, findings.HreflangInvalid.New(p.URL, invalid...))
	}
	if !hasSelf {
		out = append(out, findings.HreflangSelfMissing.New(p.URL))
	}
	return out
}

// SocialFindings reports problems with the social preview metadata of one
// page. A page without any social tags is not itself a finding; the
// "no social metadata" finding is produced by the caller when the page is
// indexable and has no tags at all.
func (p *Page) SocialFindings() []findings.Finding {
	if len(p.Social) == 0 {
		return nil
	}
	var out []findings.Finding
	og := map[string]string{}
	twitter := map[string]string{}
	var imageValues []string
	for _, s := range p.Social {
		switch {
		case strings.HasPrefix(s.Property, "og:"):
			og[s.Property] = s.Value
		case strings.HasPrefix(s.Property, "twitter:"):
			twitter[s.Property] = s.Value
		}
		if s.Property == "og:image" || s.Property == "twitter:image" {
			imageValues = append(imageValues, s.Value)
		}
	}
	// og:title/description should agree with the twitter equivalents.
	for _, prop := range []string{"og:title", "og:description"} {
		if v, ok := og[prop]; ok {
			tw := twitter[strings.Replace(prop, "og:", "twitter:", 1)]
			if tw != "" && !strings.EqualFold(tw, v) {
				out = append(out, findings.SocialInconsistent.New(p.URL, prop+": "+v, strings.Replace(prop, "og:", "twitter:", 1)+": "+tw))
			}
		}
	}
	// Every listed image must be an absolute http(s) URL.
	if len(imageValues) > 0 {
		var bad []string
		for _, v := range imageValues {
			if v == "" {
				continue
			}
			if _, err := url.ParseRequestURI(v); err != nil || !isHttpURL(v) {
				bad = append(bad, "og:image: "+findings.Truncate(v, 120))
			}
		}
		if len(bad) > 0 {
			out = append(out, findings.SocialImageInvalid.New(p.URL, bad...))
		}
	}
	return out
}

// AnchorFindings reports problems with the anchor text of a page's internal
// links.
func (p *Page) AnchorFindings() []findings.Finding {
	var out []findings.Finding
	var empty []string
	texts := map[string][]string{}
	for _, l := range p.Links {
		if l.External || l.Resource {
			continue
		}
		if l.Empty {
			empty = append(empty, "to "+shorten(l.URL, 120))
			continue
		}
		key := strings.ToLower(l.AnchorText)
		texts[key] = append(texts[key], l.URL)
	}
	if len(empty) > 0 {
		out = append(out, findings.AnchorEmpty.New(p.URL, empty...))
	}
	// Repeated anchor text pointing at different targets.
	for text, urls := range texts {
		if len(urls) < 2 {
			continue
		}
		seen := map[string]bool{}
		distinct := 0
		for _, u := range urls {
			if !seen[u] {
				seen[u] = true
				distinct++
			}
		}
		if distinct < 2 {
			continue
		}
		out = append(out, findings.AnchorDuplicate.New(p.URL,
			fmt.Sprintf("%q used for %d links to %d pages", text, len(urls), distinct)))
	}
	return out
}

// isHttpURL reports whether s is an absolute http or https URL.
func isHttpURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// shorten truncates s for display in evidence, marking truncation.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
