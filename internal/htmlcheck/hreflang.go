package htmlcheck

import (
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/language"
	"strings"
)

func (p *Page) setLang(root *html.Node) {
	walk(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == atom.Html {
			p.Lang, p.HasLang = attrOK(n, "lang")
		}
		return true
	})
}

func validHreflang(value string) bool {
	if strings.EqualFold(value, "x-default") {
		return true
	}
	parts := strings.Split(value, "-")
	if len(parts[0]) != 2 || strings.Contains(value, "_") {
		return false
	}
	if len(parts) > 3 {
		return false
	}
	base, err := language.ParseBase(parts[0])
	if err != nil || base.String() == "und" {
		return false
	}
	i := 1
	if i < len(parts) && len(parts[i]) == 4 {
		if _, err := language.ParseScript(parts[i]); err != nil {
			return false
		}
		i++
	}
	if i < len(parts) {
		region, err := language.ParseRegion(parts[i])
		if len(parts[i]) != 2 || err != nil || !region.IsCountry() || strings.EqualFold(parts[i], "UK") {
			return false
		}
		i++
	}
	return i == len(parts)
}

func (p *Page) HreflangFindings() []findings.Finding {
	var out []findings.Finding
	if len(p.Hreflangs) == 0 {
		return out
	}
	seen := map[string]string{}
	self, fallback := false, false
	for _, h := range p.Hreflangs {
		key := strings.ToLower(h.Target)
		if !validHreflang(h.Target) {
			out = append(out, findings.HreflangInvalid.New(p.URL, h.Target))
		}
		if previous, ok := seen[key]; ok && previous != h.Href {
			out = append(out, findings.HreflangConflict.New(p.URL, key, previous, h.Href))
		}
		seen[key] = h.Href
		fallback = fallback || key == "x-default"
		if h.Href == p.URL && key != "x-default" {
			self = true
			if p.HasLang && strings.Split(strings.ToLower(p.Lang), "-")[0] != strings.Split(key, "-")[0] {
				out = append(out, findings.LangMismatch.New(p.URL, p.Lang, h.Target))
			}
		}
	}
	if !self {
		out = append(out, findings.HreflangSelfMissing.New(p.URL))
	}
	if !fallback && len(seen) > 1 {
		out = append(out, findings.HreflangNoDefault.New(p.URL))
	}
	return out
}

func (p *Page) LangFindings() []findings.Finding {
	if !p.HasLang {
		return []findings.Finding{findings.LangMissing.New(p.URL)}
	}
	tag, err := language.Parse(p.Lang)
	if err != nil || tag == language.Und || strings.Contains(p.Lang, "_") {
		return []findings.Finding{findings.LangInvalid.New(p.URL, p.Lang)}
	}
	return nil
}

// HreflangSiteFindings checks only observed targets; unvisited URLs are unknown.
func HreflangSiteFindings(pages []*Page) []findings.Finding {
	byURL := map[string]*Page{}
	for _, p := range pages {
		byURL[p.URL] = p
	}
	var out []findings.Finding
	for _, p := range pages {
		for _, h := range p.Hreflangs {
			target := byURL[h.Href]
			if target == nil || target == p {
				continue
			}
			back := false
			for _, other := range target.Hreflangs {
				back = back || other.Href == p.URL
			}
			if !back {
				out = append(out, findings.HreflangNoReturn.New(p.URL, h.Href))
			}
			if !target.Robots.Indexable() {
				out = append(out, findings.HreflangTargetProblem.New(p.URL, h.Href))
			}
		}
	}
	return out
}
