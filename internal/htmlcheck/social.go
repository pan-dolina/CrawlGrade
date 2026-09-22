package htmlcheck

import (
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// MaxSocialValues bounds the social values recorded per page.
const MaxSocialValues = 10

// MaxLinks bounds the links recorded per page.
const MaxLinks = 1000

// MaxImages bounds the images recorded per page.
const MaxImages = 500

// Link is one <a> link found on a page.
type Link struct {
	Href       string // raw href as written in the document
	URL        string // resolved absolute URL; empty when unresolved or invalid
	External   bool   // points outside the page's own host
	Resource   bool   // names a non-HTML resource (by extension)
	NoFollow   bool   // rel carries nofollow, sponsored, ugc, noopener or noreferrer
	NewTab     bool   // target="_blank"
	AnchorText string // collapsed visible text, falling back to image alt text
	Empty      bool   // the anchor carried no usable text
}

// Image is one <img> found on a page.
type Image struct {
	Src      string // resolved absolute URL; empty when unresolved or invalid
	HasAlt   bool
	Width    int
	Height   int
	Loading  string
	Alt      string // collapsed alt text
	External bool   // points outside the page's own host
	Resource bool   // names a non-HTML resource (by extension)
}

// Hreflang is one rel=alternate hreflang declaration.
type Hreflang struct {
	Href   string // resolved absolute URL
	Target string // the hreflang value (for example "pl" or "en-US")
}

// Social is one social preview meta tag.
type Social struct {
	Property string // og:title, twitter:title, ...
	Value    string // collapsed content
}

// addSocial records an Open Graph or Twitter Card meta tag. Only property
// tags (og:*) and twitter: meta tags are considered.
func (p *Page) addSocial(n *html.Node) {
	if len(p.Social) >= MaxSocialValues {
		return
	}
	prop := strings.ToLower(strings.TrimSpace(attr(n, "property")))
	if strings.HasPrefix(prop, "og:") {
		p.Social = append(p.Social, Social{Property: prop, Value: clip(collapse(attr(n, "content")))})
		return
	}
	if name := strings.ToLower(strings.TrimSpace(attr(n, "name"))); name == "twitter:card" || strings.HasPrefix(name, "twitter:") {
		p.Social = append(p.Social, Social{Property: name, Value: clip(collapse(attr(n, "content")))})
	}
}

// addImage records an <img> so the link graph can flag broken and resource
// images. Images are never followed.
func (p *Page) addImage(n *html.Node) {
	if len(p.Images) >= MaxImages {
		return
	}
	src := strings.TrimSpace(attr(n, "src"))
	if src == "" {
		return
	}
	img := Image{Alt: clip(collapse(attr(n, "alt"))), Loading: strings.ToLower(attr(n, "loading"))}
	_, img.HasAlt = attrOK(n, "alt")
	img.Width, _ = strconv.Atoi(attr(n, "width"))
	img.Height, _ = strconv.Atoi(attr(n, "height"))
	if u, err := p.Resolve(src); err == nil && u.Fragment == "" && u.RawFragment == "" {
		img.Src = u.String()
		img.External = !p.scope.Contains(u)
		img.Resource = isResourceURL(u)
	}
	p.Images = append(p.Images, img)
}

// addLink records an <a> link.
func (p *Page) addLink(n *html.Node) {
	if len(p.Links) >= MaxLinks {
		return
	}
	ref := strings.TrimSpace(attr(n, "href"))
	if ref == "" {
		return
	}
	l := Link{Href: ref}
	if u, err := p.Resolve(ref); err == nil && u.Fragment == "" && u.RawFragment == "" {
		l.URL = u.String()
		l.External = !p.scope.Contains(u)
		l.Resource = isResourceURL(u)
	}
	rel := strings.Fields(attr(n, "rel"))
	l.NoFollow = slices.ContainsFunc(rel, isLinkRelToken)
	l.NewTab = strings.EqualFold(attr(n, "target"), "_blank")
	l.AnchorText = clip(collapse(anchorText(n)))
	l.Empty = l.AnchorText == ""
	p.Links = append(p.Links, l)
}

// isLinkRelToken reports whether tok is a rel token that matters for linking.
func isLinkRelToken(tok string) bool {
	switch strings.ToLower(tok) {
	case "nofollow", "sponsored", "ugc":
		return true
	}
	return false
}

// anchorText returns the visible text of an anchor, falling back to the alt
// text of a single contained image.
func anchorText(n *html.Node) string {
	if t := text(n); t != "" {
		return t
	}
	var alts []string
	walk(n, func(c *html.Node) bool {
		if c.Type == html.ElementNode && c.DataAtom == atom.Img {
			if a := collapse(attr(c, "alt")); a != "" {
				alts = append(alts, a)
			}
		}
		return true
	})
	return strings.Join(alts, " ")
}

// isResourceURL reports whether u names a non-HTML file, by extension. It
// mirrors the crawler's heuristic so the link model can flag resource links
// without importing the crawler package.
func isResourceURL(u *url.URL) bool {
	return resourceExtensions[strings.ToLower(path.Ext(u.Path))]
}

// resourceExtensions lists the file extensions that name a non-HTML resource.
// It mirrors the crawler's list so resource links are flagged consistently.
var resourceExtensions = map[string]bool{
	".7z": true, ".avi": true, ".avif": true, ".bmp": true, ".css": true, ".csv": true,
	".doc": true, ".docx": true, ".eot": true, ".exe": true, ".gif": true, ".gz": true,
	".ico": true, ".jpeg": true, ".jpg": true, ".js": true, ".json": true, ".m4a": true,
	".mov": true, ".mp3": true, ".mp4": true, ".odt": true, ".ogg": true, ".otf": true,
	".pdf": true, ".png": true, ".ppt": true, ".pptx": true, ".rar": true, ".rss": true,
	".svg": true, ".tar": true, ".tif": true, ".tiff": true, ".ttf": true, ".txt": true,
	".wav": true, ".webm": true, ".webp": true, ".woff": true, ".woff2": true,
	".xls": true, ".xlsx": true, ".xml": true, ".zip": true,
}
