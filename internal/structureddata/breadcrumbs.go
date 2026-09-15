package structureddata

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Crumb is one breadcrumb entry.
type Crumb struct {
	Position int    `json:"position"` // 0 when missing or invalid
	Name     string `json:"name"`
	Item     string `json:"item,omitempty"`
}

// Breadcrumb is a BreadcrumbList found on a page.
type Breadcrumb struct {
	Source string   `json:"source"` // "json-ld" or "microdata"
	Where  string   `json:"where"`
	Items  []Crumb  `json:"items"`
	Errors []string `json:"errors,omitempty"` // malformed structure
	Issues []string `json:"issues,omitempty"` // invalid entries
	Dups   []string `json:"duplicates,omitempty"`
}

// Breadcrumbs returns the breadcrumb lists of the page.
func (r *Result) Breadcrumbs() []Breadcrumb {
	var out []Breadcrumb
	for _, it := range r.Items {
		if it.Is("BreadcrumbList") {
			out = append(out, jsonLDBreadcrumb(it))
		}
	}
	for i, m := range r.Microdata {
		if containsType(m.Types, "BreadcrumbList") {
			out = append(out, microBreadcrumb(m, fmt.Sprintf("microdata item %d", i+1)))
		}
	}
	return out
}

func containsType(types []string, t string) bool {
	for _, x := range types {
		if x == t {
			return true
		}
	}
	return false
}

func jsonLDBreadcrumb(it Item) Breadcrumb {
	b := Breadcrumb{Source: "json-ld", Where: fmt.Sprintf("block %d %s", it.Block, it.Path)}
	var elems []any
	switch t := it.Props["itemListElement"].(type) {
	case []any:
		elems = t
	case map[string]any:
		elems = []any{t}
	case nil:
		b.Errors = append(b.Errors, "missing itemListElement")
		return b
	default:
		b.Errors = append(b.Errors, "itemListElement is not a list of ListItem objects")
		return b
	}
	if len(elems) == 0 {
		b.Errors = append(b.Errors, "itemListElement is empty")
	}
	for i, e := range elems {
		m, ok := e.(map[string]any)
		if !ok {
			b.Errors = append(b.Errors, fmt.Sprintf("itemListElement[%d] is not an object", i))
			continue
		}
		if !containsType(typesOf(m["@type"]), "ListItem") {
			b.Errors = append(b.Errors, fmt.Sprintf("itemListElement[%d] is not a ListItem", i))
			continue
		}
		c := Crumb{}
		c.Position, ok = intValue(m["position"])
		if !ok {
			b.Issues = append(b.Issues, fmt.Sprintf("itemListElement[%d]: position %s is not a positive integer", i, describe(m["position"])))
		}
		c.Name = stringValue(m["name"])
		switch item := m["item"].(type) {
		case string:
			c.Item = strings.TrimSpace(item)
		case map[string]any:
			c.Item = firstNonEmpty(stringValue(item["@id"]), stringValue(item["url"]))
			if c.Name == "" {
				c.Name = stringValue(item["name"])
			}
		}
		b.Items = append(b.Items, c)
	}
	b.validate()
	return b
}

func microBreadcrumb(m MicroItem, where string) Breadcrumb {
	b := Breadcrumb{Source: "microdata", Where: where}
	elems := m.Props["itemListElement"]
	if len(elems) == 0 {
		b.Errors = append(b.Errors, "missing itemListElement")
		return b
	}
	for i, v := range elems {
		if v.Item == nil || !containsType(v.Item.Types, "ListItem") {
			b.Errors = append(b.Errors, fmt.Sprintf("itemListElement %d is not an itemscope of type ListItem", i+1))
			continue
		}
		props := v.Item.Props
		c := Crumb{}
		var ok bool
		if pos := first(props["position"]); pos != nil {
			c.Position, ok = intValue(pos.Text)
		}
		if !ok {
			b.Issues = append(b.Issues, fmt.Sprintf("itemListElement %d: missing or invalid position", i+1))
		}
		if n := first(props["name"]); n != nil {
			c.Name = n.Text
		}
		if it := first(props["item"]); it != nil {
			c.Item = it.URL
			if c.Name == "" && it.Item != nil {
				if n := first(it.Item.Props["name"]); n != nil {
					c.Name = n.Text
				}
			}
		}
		b.Items = append(b.Items, c)
	}
	b.validate()
	return b
}

func first(vs []MicroValue) *MicroValue {
	if len(vs) == 0 {
		return nil
	}
	return &vs[0]
}

// validate checks names, items and duplicates. The last entry may omit
// item: it represents the current page.
func (b *Breadcrumb) validate() {
	positions := map[int]int{}
	items := map[string]int{}
	for i, c := range b.Items {
		label := fmt.Sprintf("entry %d", i+1)
		if c.Name == "" {
			b.Issues = append(b.Issues, label+": missing name")
		}
		if c.Item == "" && i != len(b.Items)-1 {
			b.Issues = append(b.Issues, label+": missing item URL")
		} else if c.Item != "" && !validURLValue(c.Item) {
			b.Issues = append(b.Issues, fmt.Sprintf("%s: item %q is not a valid URL", label, findings.Truncate(c.Item, 100)))
		}
		if c.Position > 0 {
			if prev, dup := positions[c.Position]; dup {
				b.Dups = append(b.Dups, fmt.Sprintf("position %d used by entries %d and %d", c.Position, prev, i+1))
			} else {
				positions[c.Position] = i + 1
			}
		}
		if c.Item != "" {
			if prev, dup := items[c.Item]; dup {
				b.Dups = append(b.Dups, fmt.Sprintf("item %s used by entries %d and %d", findings.Truncate(c.Item, 100), prev, i+1))
			} else {
				items[c.Item] = i + 1
			}
		}
	}
	if len(b.Dups) == 0 && len(positions) == len(b.Items) {
		for want := 1; want <= len(b.Items); want++ {
			if _, ok := positions[want]; !ok {
				b.Issues = append(b.Issues, fmt.Sprintf("positions are not sequential from 1 (missing %d)", want))
				break
			}
		}
	}
}

func intValue(v any) (int, bool) {
	var n int
	var err error
	switch t := v.(type) {
	case json.Number:
		n, err = strconv.Atoi(t.String())
	case string:
		n, err = strconv.Atoi(strings.TrimSpace(t))
	default:
		return 0, false
	}
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func describe(v any) string {
	if v == nil {
		return "(missing)"
	}
	b, _ := json.Marshal(v)
	return findings.Truncate(string(b), 40)
}

// BreadcrumbFindings reports breadcrumb problems for the page.
func (r *Result) BreadcrumbFindings(pageURL string) []findings.Finding {
	var malformed, invalid, dups []string
	for _, b := range r.Breadcrumbs() {
		for _, e := range b.Errors {
			malformed = append(malformed, fmt.Sprintf("%s %s: %s", b.Source, b.Where, e))
		}
		for _, e := range b.Issues {
			invalid = append(invalid, fmt.Sprintf("%s %s: %s", b.Source, b.Where, e))
		}
		for _, e := range b.Dups {
			dups = append(dups, fmt.Sprintf("%s %s: %s", b.Source, b.Where, e))
		}
	}
	var out []findings.Finding
	if len(malformed) > 0 {
		out = append(out, findings.BreadcrumbMalformed.New(pageURL, malformed...))
	}
	if len(invalid) > 0 {
		out = append(out, findings.BreadcrumbInvalid.New(pageURL, invalid...))
	}
	if len(dups) > 0 {
		out = append(out, findings.BreadcrumbDuplicate.New(pageURL, dups...))
	}
	return out
}
