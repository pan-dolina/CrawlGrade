package structureddata

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Microdata limits.
const (
	MaxMicroItems = 1000
	maxMicroText  = 512
)

// MicroItem is an element with itemscope.
type MicroItem struct {
	Types []string                `json:"types"`
	Props map[string][]MicroValue `json:"-"`
}

// MicroValue is the value of one itemprop.
type MicroValue struct {
	Text string     // text or attribute value
	URL  string     // resolved URL for URL-valued elements (a, link, img, ...)
	Item *MicroItem // nested item when the property element has itemscope
}

// String returns the textual value, preferring a URL.
func (v MicroValue) String() string {
	if v.URL != "" {
		return v.URL
	}
	return v.Text
}

// extractMicrodata returns every microdata item in document order. Nested
// items are returned both as property values and in the list.
func extractMicrodata(root *html.Node, base *url.URL) []MicroItem {
	var items []*MicroItem
	byNode := map[*html.Node]*MicroItem{}
	walk(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Namespace != "" {
			return n.Type != html.ElementNode
		}
		if n.DataAtom == atom.Template {
			return false
		}
		if hasAttr(n, "itemscope") && len(items) < MaxMicroItems {
			it := &MicroItem{Types: typesOfList(attr(n, "itemtype")), Props: map[string][]MicroValue{}}
			items = append(items, it)
			byNode[n] = it
		}
		return true
	})
	// Assign properties to the nearest ancestor item. An element that is
	// itself an item belongs, as a property value, to its parent item.
	walk(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		if n.DataAtom == atom.Template {
			return false
		}
		props := strings.Fields(attr(n, "itemprop"))
		if len(props) == 0 {
			return true
		}
		owner := nearestItem(n.Parent, byNode)
		if owner == nil {
			return true
		}
		v := microValue(n, base, byNode)
		for _, p := range props {
			if len(owner.Props[p]) < MaxMicroItems {
				owner.Props[p] = append(owner.Props[p], v)
			}
		}
		return true
	})
	out := make([]MicroItem, len(items))
	for i, it := range items {
		out[i] = *it
	}
	return out
}

func nearestItem(n *html.Node, byNode map[*html.Node]*MicroItem) *MicroItem {
	for ; n != nil; n = n.Parent {
		if it, ok := byNode[n]; ok {
			return it
		}
	}
	return nil
}

func microValue(n *html.Node, base *url.URL, byNode map[*html.Node]*MicroItem) MicroValue {
	if it, ok := byNode[n]; ok {
		v := MicroValue{Item: it}
		if id := attr(n, "itemid"); id != "" {
			v.URL = resolve(base, id)
		} else if href := attr(n, "href"); href != "" {
			v.URL = resolve(base, href)
		}
		return v
	}
	switch n.DataAtom {
	case atom.Meta:
		return MicroValue{Text: attr(n, "content")}
	case atom.A, atom.Area, atom.Link:
		return MicroValue{URL: resolve(base, attr(n, "href"))}
	case atom.Img, atom.Audio, atom.Video, atom.Source, atom.Iframe, atom.Embed, atom.Track:
		return MicroValue{URL: resolve(base, attr(n, "src"))}
	case atom.Object:
		return MicroValue{URL: resolve(base, attr(n, "data"))}
	case atom.Data, atom.Meter:
		return MicroValue{Text: attr(n, "value")}
	case atom.Time:
		if dt, ok := attrOK(n, "datetime"); ok {
			return MicroValue{Text: dt}
		}
	}
	return MicroValue{Text: textContent(n)}
}

func resolve(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if base == nil {
		return ref
	}
	u, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

func textContent(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode && b.Len() < 4*maxMicroText {
			b.WriteString(c.Data)
			b.WriteByte(' ')
		}
		return true
	})
	s := strings.Join(strings.Fields(b.String()), " ")
	if len(s) > maxMicroText {
		s = strings.ToValidUTF8(s[:maxMicroText], "")
	}
	return s
}

func typesOfList(itemtype string) []string {
	var raw []any
	for _, f := range strings.Fields(itemtype) {
		raw = append(raw, f)
	}
	return typesOf(raw)
}

func hasAttr(n *html.Node, key string) bool {
	_, ok := attrOK(n, key)
	return ok
}

func attrOK(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}
