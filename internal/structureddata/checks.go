package structureddata

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

// Recognized lists the schema.org types CrawlGrade checks, after aliases are
// resolved.
var Recognized = []string{"Article", "BreadcrumbList", "FAQPage", "LocalBusiness", "Organization", "Product", "WebPage", "WebSite"}

// aliases maps common subtypes to the recognized type whose checks apply.
var aliases = map[string]string{
	"NewsArticle": "Article", "BlogPosting": "Article", "TechArticle": "Article", "Report": "Article",
	"Store": "LocalBusiness", "Optician": "LocalBusiness", "Restaurant": "LocalBusiness",
	"MedicalBusiness": "LocalBusiness", "ProfessionalService": "LocalBusiness", "AutomotiveBusiness": "LocalBusiness",
	"HealthAndBeautyBusiness": "LocalBusiness", "LegalService": "LocalBusiness", "FinancialService": "LocalBusiness",
	"Corporation": "Organization", "NGO": "Organization", "EducationalOrganization": "Organization",
	"AboutPage": "WebPage", "ContactPage": "WebPage", "CollectionPage": "WebPage", "ItemPage": "WebPage",
	"ProfilePage": "WebPage", "SearchResultsPage": "WebPage", "CheckoutPage": "WebPage", "FAQPage": "FAQPage",
}

// Canonical returns the recognized type t belongs to, or "".
func Canonical(t string) string {
	if a, ok := aliases[t]; ok {
		return a
	}
	if slices.Contains(Recognized, t) {
		return t
	}
	return ""
}

type typeRule struct {
	// required lists groups of alternatives; each group needs one property.
	required    [][]string
	recommended []string
}

var typeRules = map[string]typeRule{
	"Organization":  {required: [][]string{{"name"}}, recommended: []string{"url", "logo"}},
	"LocalBusiness": {required: [][]string{{"name"}, {"address"}}, recommended: []string{"telephone", "openingHoursSpecification", "url"}},
	"WebSite":       {required: [][]string{{"name", "url"}}, recommended: []string{"url"}},
	"WebPage":       {required: [][]string{{"name", "url", "headline"}}},
	"Article":       {required: [][]string{{"headline"}}, recommended: []string{"author", "datePublished", "image"}},
	"Product":       {required: [][]string{{"name"}, {"offers", "review", "aggregateRating"}}, recommended: []string{"image", "description"}},
	// BreadcrumbList has dedicated validation in breadcrumbs.go.
	"FAQPage": {required: [][]string{{"mainEntity"}}},
}

var urlProps = []string{"url", "logo", "image", "sameAs", "item"}

func present(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(t) != ""
	case []any:
		return slices.ContainsFunc(t, present)
	case map[string]any:
		return len(t) > 0
	}
	return true
}

// Findings returns structured data findings for the page at pageURL.
func (r *Result) Findings(pageURL string) []findings.Finding {
	var out []findings.Finding
	if len(r.Errors) > 0 {
		var ev []string
		for _, e := range r.Errors {
			ev = append(ev, fmt.Sprintf("block %d: %s", e.Block, e.Message))
		}
		out = append(out, findings.SchemaInvalid.New(pageURL, ev...))
	}
	if len(r.Untyped) > 0 {
		out = append(out, findings.SchemaUntyped.New(pageURL, r.Untyped...))
	}
	if len(r.ContextIssues) > 0 {
		out = append(out, findings.SchemaContext.New(pageURL, r.ContextIssues...))
	}
	var required, recommended, badURLs []string
	for _, it := range r.Items {
		checked := map[string]bool{}
		for _, t := range it.Types {
			ct := Canonical(t)
			rule, ok := typeRules[ct]
			if !ok || checked[ct] {
				continue
			}
			checked[ct] = true
			where := fmt.Sprintf("block %d %s (%s)", it.Block, it.Path, t)
			for _, group := range rule.required {
				if !slices.ContainsFunc(group, func(p string) bool { return present(it.Props[p]) }) {
					required = append(required, fmt.Sprintf("%s: missing %s", where, strings.Join(group, " or ")))
				}
			}
			if ct == "FAQPage" {
				required = append(required, faqIssues(it, where)...)
			}
			// Nested entities (an author, a publisher) are not expected to
			// carry recommended properties.
			if !it.TopLevel {
				continue
			}
			var missing []string
			for _, p := range rule.recommended {
				if !present(it.Props[p]) {
					missing = append(missing, p)
				}
			}
			if len(missing) > 0 {
				recommended = append(recommended, fmt.Sprintf("%s: %s", where, strings.Join(missing, ", ")))
			}
		}
		for _, p := range urlProps {
			for _, s := range stringValues(it.Props[p]) {
				if !validURLValue(s) {
					badURLs = append(badURLs, fmt.Sprintf("block %d %s.%s: %s", it.Block, it.Path, p, findings.Truncate(s, 120)))
				}
			}
		}
	}
	if len(required) > 0 {
		out = append(out, findings.SchemaRequired.New(pageURL, required...))
	}
	if len(recommended) > 0 {
		out = append(out, findings.SchemaRecommended.New(pageURL, recommended...))
	}
	if len(badURLs) > 0 {
		out = append(out, findings.SchemaInvalidURL.New(pageURL, badURLs...))
	}
	if r.Truncated {
		out = append(out, findings.SchemaLimits.New(pageURL, fmt.Sprintf("blocks: %d, items: %d", r.Blocks, len(r.Items))))
	}
	return out
}

func faqIssues(it Item, where string) []string {
	var out []string
	var questions []any
	switch t := it.Props["mainEntity"].(type) {
	case []any:
		questions = t
	case map[string]any:
		questions = []any{t}
	}
	for i, q := range questions {
		qm, ok := q.(map[string]any)
		if !ok || !slices.Contains(typesOf(qm["@type"]), "Question") {
			out = append(out, fmt.Sprintf("%s: mainEntity[%d] is not a Question", where, i))
			continue
		}
		if !present(qm["name"]) {
			out = append(out, fmt.Sprintf("%s: mainEntity[%d] missing name", where, i))
		}
		ans, _ := qm["acceptedAnswer"].(map[string]any)
		if ans == nil || !present(ans["text"]) {
			out = append(out, fmt.Sprintf("%s: mainEntity[%d] missing acceptedAnswer.text", where, i))
		}
	}
	return out
}

func stringValues(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		var out []string
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// validURLValue accepts absolute http(s) URLs and relative references, which
// JSON-LD resolves against the document URL.
func validURLValue(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		return !strings.ContainsAny(s, " \t\n")
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
