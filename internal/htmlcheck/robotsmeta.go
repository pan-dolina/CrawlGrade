package htmlcheck

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// maxRobotsSources bounds recorded robots meta tags and headers per page.
const maxRobotsSources = 20

// RobotsSource is one robots meta tag or X-Robots-Tag header value.
type RobotsSource struct {
	Source string `json:"source"` // "meta" or "header"
	Agent  string `json:"agent"`  // "robots" for all crawlers, or a crawler name
	Value  string `json:"value"`
}

// Robots is the combined robots metadata of a page. Directives from all
// sources that apply to generic crawlers or to Googlebot are merged, and
// the most restrictive value wins, as search engines document.
type Robots struct {
	Sources          []RobotsSource `json:"sources,omitempty"`
	NoIndex          bool           `json:"noindex"`
	NoFollow         bool           `json:"nofollow"`
	NoArchive        bool           `json:"noarchive,omitempty"`
	NoSnippet        bool           `json:"nosnippet,omitempty"`
	NoImageIndex     bool           `json:"noimageindex,omitempty"`
	MaxSnippet       *int           `json:"max_snippet,omitempty"`
	MaxImagePreview  string         `json:"max_image_preview,omitempty"`
	MaxVideoPreview  *int           `json:"max_video_preview,omitempty"`
	UnavailableAfter string         `json:"unavailable_after,omitempty"`

	explicitIndex, explicitFollow bool
	invalid                       []string
}

// robotsAgents are the meta names treated as robots directives. Directives
// for other named crawlers are recorded but do not change the result.
var robotsAgents = []string{"robots", "googlebot", "googlebot-news", "bingbot"}

func isAppliedAgent(agent string) bool {
	return agent == "robots" || agent == "googlebot"
}

func (p *Page) addRobots(source, agent, value string) {
	if len(p.Robots.Sources) >= maxRobotsSources {
		return
	}
	value = clip(collapse(value))
	p.Robots.Sources = append(p.Robots.Sources, RobotsSource{Source: source, Agent: agent, Value: value})
	if isAppliedAgent(agent) {
		p.Robots.apply(value)
	}
}

// AddRobotsHeaders records X-Robots-Tag header values. A value may start
// with a crawler name ("googlebot: noindex").
func (p *Page) AddRobotsHeaders(values []string) {
	for _, v := range values {
		agent := "robots"
		if name, rest, ok := strings.Cut(v, ":"); ok {
			n := strings.ToLower(strings.TrimSpace(name))
			if !isDirective(n) && !strings.ContainsAny(n, " ,") {
				agent, v = n, rest
			}
		}
		p.addRobots("header", agent, v)
	}
}

func isDirective(name string) bool {
	switch name {
	case "max-snippet", "max-image-preview", "max-video-preview", "unavailable_after":
		return true
	}
	return false
}

func (r *Robots) apply(value string) {
	for _, raw := range strings.Split(value, ",") {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			continue
		}
		key, arg, hasArg := strings.Cut(tok, ":")
		key = strings.ToLower(strings.TrimSpace(key))
		arg = strings.TrimSpace(arg)
		switch key {
		case "index":
			r.explicitIndex = true
		case "noindex":
			r.NoIndex = true
		case "follow":
			r.explicitFollow = true
		case "nofollow":
			r.NoFollow = true
		case "none":
			r.NoIndex, r.NoFollow = true, true
		case "all":
			r.explicitIndex, r.explicitFollow = true, true
		case "noarchive", "nocache":
			r.NoArchive = true
		case "nosnippet":
			r.NoSnippet = true
		case "noimageindex":
			r.NoImageIndex = true
		case "notranslate", "indexifembedded", "noodp", "noydir":
		case "max-snippet", "max-video-preview":
			n, err := strconv.Atoi(arg)
			if !hasArg || err != nil || n < -1 {
				r.invalid = append(r.invalid, tok)
				continue
			}
			target := &r.MaxSnippet
			if key == "max-video-preview" {
				target = &r.MaxVideoPreview
			}
			// Most restrictive wins; -1 means no limit.
			if *target == nil || (n != -1 && (**target == -1 || n < **target)) {
				v := n
				*target = &v
			}
		case "max-image-preview":
			v := strings.ToLower(arg)
			rank := map[string]int{"none": 0, "standard": 1, "large": 2}
			nr, ok := rank[v]
			if !hasArg || !ok {
				r.invalid = append(r.invalid, tok)
				continue
			}
			if cur, set := rank[r.MaxImagePreview]; r.MaxImagePreview == "" || (set && nr < cur) {
				r.MaxImagePreview = v
			}
		case "unavailable_after":
			if !hasArg || arg == "" {
				r.invalid = append(r.invalid, tok)
				continue
			}
			r.UnavailableAfter = arg
		default:
			r.invalid = append(r.invalid, tok)
		}
	}
}

// Indexable reports whether robots metadata allows indexing.
func (r *Robots) Indexable() bool { return !r.NoIndex }

// RobotsFindings checks robots metadata of one page. start is true for the
// audited start URL, where noindex is critical.
func (p *Page) RobotsFindings(start bool) []findings.Finding {
	r := &p.Robots
	var out []findings.Finding
	var ev []string
	for _, s := range r.Sources {
		ev = append(ev, fmt.Sprintf("%s %s: %s", s.Source, s.Agent, s.Value))
	}
	if r.NoIndex {
		if start {
			out = append(out, findings.IndexStartNoindex.New(p.URL, ev...))
		} else {
			out = append(out, findings.IndexNoindex.New(p.URL, ev...))
		}
	}
	if r.NoFollow {
		out = append(out, findings.IndexNofollow.New(p.URL, ev...))
	}
	if (r.NoIndex && r.explicitIndex) || (r.NoFollow && r.explicitFollow) {
		out = append(out, findings.IndexConflict.New(p.URL, ev...))
	}
	if len(r.invalid) > 0 {
		out = append(out, findings.IndexInvalidDirective.New(p.URL, quoteAll(r.invalid)...))
	}
	var restrictions []string
	if r.NoSnippet {
		restrictions = append(restrictions, "nosnippet")
	}
	if r.MaxSnippet != nil && *r.MaxSnippet == 0 {
		restrictions = append(restrictions, "max-snippet:0")
	}
	if r.MaxImagePreview == "none" {
		restrictions = append(restrictions, "max-image-preview:none")
	}
	if r.NoArchive {
		restrictions = append(restrictions, "noarchive")
	}
	if len(restrictions) > 0 {
		out = append(out, findings.IndexSnippetRestricted.New(p.URL, restrictions...))
	}
	if r.NoIndex {
		if c := p.CanonicalURL(); c != "" && c != p.URL {
			out = append(out, findings.IndexNoindexCanonical.New(p.URL, "canonical: "+c))
		}
	}
	return out
}

func robotsAgent(name string) (string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	return name, slices.Contains(robotsAgents, name)
}
