package structureddata

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
	"golang.org/x/net/html"
)

func extract(t *testing.T, doc string) *Result {
	t.Helper()
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return Extract(root, nil)
}

func script(body string) string {
	return `<script type="application/ld+json">` + body + `</script>`
}

func ids(fs []findings.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.ID)
	}
	return strings.Join(s, ",")
}

func TestRecognizedTypes(t *testing.T) {
	doc := script(`{"@context":"https://schema.org","@graph":[
		{"@type":"Organization","name":"Salon","url":"https://example.com/","logo":"https://example.com/logo.png"},
		{"@type":"WebSite","name":"Salon","url":"https://example.com/"},
		{"@type":"WebPage","name":"Home"}
	]}`) + script(`[{"@context":"http://schema.org/","@type":["LocalBusiness","Optician"],"name":"Salon","address":"ul. Testowa 1","telephone":"1","openingHoursSpecification":{"opens":"09:00"},"url":"/"}]`) +
		script(`{"@context":{"@vocab":"https://schema.org/"},"@type":"schema:Product","name":"Okulary","offers":{"@type":"Offer","price":"100"},"image":"a.jpg","description":"d"}`) +
		script(`{"@context":"https://schema.org","@type":"https://schema.org/Article","headline":"H","author":{"@type":"Person","name":"A"},"datePublished":"2026-01-01","image":["https://example.com/a.jpg"]}`) +
		script(`{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[{"@type":"Question","name":"Q","acceptedAnswer":{"@type":"Answer","text":"A"}}]}`) +
		script(`{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Home","item":"https://example.com/"}]}`)
	r := extract(t, doc)
	want := "Answer,Article,BreadcrumbList,FAQPage,ListItem,LocalBusiness,Offer,Optician,Organization,Person,Product,Question,WebPage,WebSite"
	if got := strings.Join(r.Types(), ","); got != want {
		t.Errorf("types = %s", got)
	}
	if fs := r.Findings("https://example.com/"); len(fs) != 0 {
		t.Errorf("unexpected findings: %+v", fs)
	}
	if r.Blocks != 6 {
		t.Errorf("blocks = %d", r.Blocks)
	}
}

func TestJSONLDProblems(t *testing.T) {
	cases := []struct {
		name, doc, want string
	}{
		{"syntax", script(`{"@context":"https://schema.org","@type":"Product", "name": "Okulary" `), "SEO-SCHEMA-001"},
		{"trailing comma", script(`{"@type":"Thing",}`), "SEO-SCHEMA-001"},
		{"extra data", script(`{"@context":"https://schema.org","@type":"Thing"} {"x":1}`), "SEO-SCHEMA-001"},
		{"empty", script(`   `), "SEO-SCHEMA-001"},
		{"scalar", script(`"just a string"`), "SEO-SCHEMA-001"},
		{"untyped", script(`{"@context":"https://schema.org","name":"x"}`), "SEO-SCHEMA-002"},
		{"no context", script(`{"@type":"Thing","name":"x"}`), "SEO-SCHEMA-003"},
		{"other context", script(`{"@context":"https://example.org/vocab","@type":"Thing"}`), "SEO-SCHEMA-003"},
		{"graph no context", script(`{"@graph":[{"@type":"Thing"}]}`), "SEO-SCHEMA-003"},
		{"product required", script(`{"@context":"https://schema.org","@type":"Product","image":"a.jpg","description":"d"}`), "SEO-SCHEMA-004"},
		{"faq answers", script(`{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[{"@type":"Question","name":"Q"},{"@type":"Answer"},"x"]}`), "SEO-SCHEMA-004"},
		{"article recommended", script(`{"@context":"https://schema.org","@type":"BlogPosting","headline":"H"}`), "SEO-SCHEMA-005"},
		{"nested author not checked", script(`{"@context":"https://schema.org","@type":"Article","headline":"H","author":"A","datePublished":"x","image":"i","publisher":{"@type":"Organization","name":"P"}}`), ""},
		{"nested org missing name", script(`{"@context":"https://schema.org","@type":"Article","headline":"H","author":"A","datePublished":"x","image":"i","publisher":{"@type":"Organization"}}`), "SEO-SCHEMA-004"},
		{"bad urls", script(`{"@context":"https://schema.org","@type":"Organization","name":"x","url":"javascript:alert(1)","logo":"data:image/png;base64,AA","sameAs":["https://ok.example/","not a url with spaces"]}`), "SEO-SCHEMA-006"},
		{"unknown type ok", script(`{"@context":"https://schema.org","@type":"Event","name":"x"}`), ""},
		{"wrong script type ignored", `<script type="application/json">{bad</script><script type="Application/LD+JSON; charset=utf-8">{"@context":"https://schema.org","@type":"Thing"}</script>`, ""},
		{"template ignored", `<template>` + script(`{bad`) + `</template>`, ""},
		{"id reference is not untyped", script(`{"@context":"https://schema.org","@id":"https://example.com/#org"}`), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := extract(t, c.doc)
			if got := ids(r.Findings("https://example.com/")); got != c.want {
				t.Errorf("findings = %q, want %q (errors %+v, items %d)", got, c.want, r.Errors, len(r.Items))
			}
		})
	}
}

func TestJSONLDErrorLineNumber(t *testing.T) {
	r := extract(t, script("{\n\"@type\": \"Thing\",\n\"name\": oops\n}"))
	if len(r.Errors) != 1 || !strings.Contains(r.Errors[0].Message, "line 3") {
		t.Errorf("errors = %+v", r.Errors)
	}
}

func TestJSONLDLimits(t *testing.T) {
	var b strings.Builder
	for i := range MaxBlocks + 3 {
		b.WriteString(script(fmt.Sprintf(`{"@context":"https://schema.org","@type":"Thing","name":"%d"}`, i)))
	}
	r := extract(t, b.String())
	if r.Blocks != MaxBlocks || !r.Truncated || ids(r.Findings("u")) != "SEO-SCHEMA-007" {
		t.Errorf("blocks=%d truncated=%v findings=%s", r.Blocks, r.Truncated, ids(r.Findings("u")))
	}

	big := script(`{"@context":"https://schema.org","@type":"Thing","name":"` + strings.Repeat("a", MaxBlockBytes) + `"}`)
	r = extract(t, big)
	if len(r.Errors) != 1 || !r.Truncated {
		t.Errorf("big block: %+v", r.Errors)
	}

	deep := `{"@context":"https://schema.org","@type":"Thing","x":` + strings.Repeat(`{"@type":"Thing","x":`, 100) + `1` + strings.Repeat("}", 100) + "}"
	r = extract(t, script(deep))
	if !r.Truncated || len(r.Items) > MaxDepth+2 {
		t.Errorf("deep: truncated=%v items=%d", r.Truncated, len(r.Items))
	}

	var many strings.Builder
	many.WriteString(`{"@context":"https://schema.org","@type":"ItemList","itemListElement":[`)
	for i := range MaxNodes + 10 {
		if i > 0 {
			many.WriteString(",")
		}
		many.WriteString(`{"@type":"Thing"}`)
	}
	many.WriteString("]}")
	r = extract(t, script(many.String()))
	if !r.Truncated || len(r.Items) > MaxNodes {
		t.Errorf("many: truncated=%v items=%d", r.Truncated, len(r.Items))
	}
}

func TestScriptContentIsNotHTMLDecoded(t *testing.T) {
	// Script content is raw text: entities stay literal and "</script>"
	// inside a JSON string must be written as "<\/script>".
	r := extract(t, script(`{"@context":"https://schema.org","@type":"Organization","name":"A &amp; B <\/script>"}`))
	if len(r.Items) != 1 || r.Items[0].Props["name"] != "A &amp; B </script>" {
		t.Errorf("items = %+v", r.Items)
	}
}

func TestTypesOfAndContext(t *testing.T) {
	if got := typesOf([]any{"schema:Product", "https://schema.org/Product", "Offer", 5, ""}); fmt.Sprint(got) != "[Product Offer]" {
		t.Errorf("typesOf = %v", got)
	}
	if schemaContext(42) || !schemaContext([]any{"https://example.org", "https://schema.org/"}) {
		t.Error("schemaContext")
	}
	if Canonical("NewsArticle") != "Article" || Canonical("Product") != "Product" || Canonical("Event") != "" {
		t.Error("Canonical")
	}
}
