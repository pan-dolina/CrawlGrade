package htmlcheck

import "github.com/pan-dolina/crawlgrade/internal/findings"

func (p *Page) ImageFindings() []findings.Finding {
	var out []findings.Finding
	for _, img := range p.Images {
		if !img.HasAlt {
			out = append(out, findings.ImageAlt.New(p.URL, img.Src))
		}
		if img.Width <= 0 || img.Height <= 0 {
			out = append(out, findings.ImageDimensions.New(p.URL, img.Src))
		}
		if img.Loading != "" && img.Loading != "lazy" && img.Loading != "eager" {
			out = append(out, findings.ImageLoading.New(p.URL, img.Src, img.Loading))
		}
	}
	return out
}

func (p *Page) SocialRequiredFindings() []findings.Finding {
	m := map[string]string{}
	og, tw := false, false
	for _, s := range p.Social {
		m[s.Property] = s.Value
		og = og || len(s.Property) > 3 && s.Property[:3] == "og:"
		tw = tw || len(s.Property) > 8 && s.Property[:8] == "twitter:"
	}
	var missing []string
	if og {
		for _, key := range []string{"og:title", "og:type", "og:url", "og:image"} {
			if m[key] == "" {
				missing = append(missing, key)
			}
		}
	}
	if tw {
		if m["twitter:card"] == "" {
			missing = append(missing, "twitter:card")
		}
		if m["twitter:title"] == "" && m["og:title"] == "" {
			missing = append(missing, "twitter:title or og:title")
		}
		if m["twitter:image"] == "" && m["og:image"] == "" {
			missing = append(missing, "twitter:image or og:image")
		}
	}
	if len(missing) > 0 {
		return []findings.Finding{findings.SocialRequired.New(p.URL, missing...)}
	}
	return nil
}
