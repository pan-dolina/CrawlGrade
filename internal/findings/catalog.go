package findings

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// catalog holds every rule by ID. Rules are registered from catalog_*.go
// files grouped by area. IDs are never renumbered or reused.
var catalog = map[string]Rule{}

var idPattern = regexp.MustCompile(`^(SEO-[A-Z]+|WEB-HYGIENE)-\d{3}$`)

func register(r Rule) Rule {
	if !idPattern.MatchString(r.ID) {
		panic(fmt.Sprintf("findings: invalid rule ID %q", r.ID))
	}
	if _, dup := catalog[r.ID]; dup {
		panic(fmt.Sprintf("findings: duplicate rule ID %q", r.ID))
	}
	if !r.Category.Valid() || !r.Severity.Valid() || r.Title == "" || r.Recommendation == "" {
		panic(fmt.Sprintf("findings: incomplete rule %q", r.ID))
	}
	if (r.Category == CategoryWebHygiene) != strings.HasPrefix(r.ID, "WEB-HYGIENE-") {
		panic(fmt.Sprintf("findings: rule %q: web hygiene IDs and category must match", r.ID))
	}
	catalog[r.ID] = r
	return r
}

// Lookup returns the rule with the given ID.
func Lookup(id string) (Rule, bool) {
	r, ok := catalog[id]
	return r, ok
}

// All returns every rule sorted by ID.
func All() []Rule {
	out := make([]Rule, 0, len(catalog))
	for _, r := range catalog {
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Rule) int { return strings.Compare(a.ID, b.ID) })
	return out
}
