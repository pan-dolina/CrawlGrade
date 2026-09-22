package structureddata

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func extractAt(t *testing.T, base, doc string) *Result {
	t.Helper()
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(base)
	return Extract(root, u)
}

func TestJSONLDBreadcrumbs(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"valid", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":1,"name":"Home","item":"https://example.com/"},
			{"@type":"ListItem","position":"2","item":{"@id":"https://example.com/shop","name":"Shop"}},
			{"@type":"ListItem","position":3,"name":"Current"}]}`, ""},
		{"missing list", `{"@context":"https://schema.org","@type":"BreadcrumbList"}`, "SEO-BREADCRUMB-001"},
		{"empty list", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[]}`, "SEO-BREADCRUMB-001"},
		{"scalar list", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":"Home > Shop"}`, "SEO-BREADCRUMB-001"},
		{"not list items", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"Thing","name":"x"},"y"]}`, "SEO-BREADCRUMB-001"},
		{"missing name and item", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":1},{"@type":"ListItem","position":2,"name":"Last"}]}`, "SEO-BREADCRUMB-002"},
		{"bad position", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":"one","name":"A","item":"https://example.com/"},{"@type":"ListItem","position":0,"name":"B"}]}`, "SEO-BREADCRUMB-002"},
		{"not sequential", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":2,"name":"A","item":"https://example.com/"},{"@type":"ListItem","position":3,"name":"B"}]}`, "SEO-BREADCRUMB-002"},
		{"bad url", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":1,"name":"A","item":"javascript:alert(1)"},{"@type":"ListItem","position":2,"name":"B"}]}`, "SEO-SCHEMA-006,SEO-BREADCRUMB-002"},
		{"duplicates", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":1,"name":"A","item":"https://example.com/"},
			{"@type":"ListItem","position":1,"name":"B","item":"https://example.com/"}]}`, "SEO-BREADCRUMB-003"},
		{"single object element", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":{"@type":"ListItem","position":1,"name":"Only"}}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := extractAt(t, "https://example.com/p", script(c.body))
			fs := append(r.Findings("https://example.com/p"), r.BreadcrumbFindings("https://example.com/p")...)
			if got := ids(fs); got != c.want {
				t.Errorf("findings = %q, want %q (%+v)", got, c.want, r.Breadcrumbs())
			}
		})
	}
}

func TestMicrodataBreadcrumbs(t *testing.T) {
	valid := `<ol itemscope itemtype="https://schema.org/BreadcrumbList">
<li itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem">
  <a itemprop="item" href="/"><span itemprop="name">Home</span></a><meta itemprop="position" content="1"></li>
<li itemprop="itemListElement" itemscope itemtype="http://schema.org/ListItem">
  <a itemprop="item" itemscope itemtype="https://schema.org/WebPage" itemid="/shop" href="/shop"><span itemprop="name">Shop</span></a>
  <meta itemprop="position" content="2"></li>
<li itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem"><span itemprop="name">Current</span><meta itemprop="position" content="3"></li>
</ol>`
	r := extractAt(t, "https://example.com/shop/item", valid)
	bs := r.Breadcrumbs()
	if len(bs) != 1 || len(bs[0].Items) != 3 {
		t.Fatalf("breadcrumbs = %+v", bs)
	}
	if bs[0].Items[0].Item != "https://example.com/" || bs[0].Items[1].Item != "https://example.com/shop" || bs[0].Items[1].Name != "Shop" {
		t.Errorf("items = %+v", bs[0].Items)
	}
	if got := ids(r.BreadcrumbFindings("u")); got != "" {
		t.Errorf("valid microdata findings = %s (%+v)", got, bs)
	}

	cases := map[string]string{
		`<div itemscope itemtype="https://schema.org/BreadcrumbList"><span itemprop="itemListElement">Brak ListItem</span></div>`: "SEO-BREADCRUMB-001",
		`<div itemscope itemtype="https://schema.org/BreadcrumbList"></div>`:                                                      "SEO-BREADCRUMB-001",
		`<div itemscope itemtype="https://schema.org/BreadcrumbList"><div itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem"><a itemprop="item" href="/">x</a></div><div itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem"><span itemprop="name">B</span><meta itemprop="position" content="2"></div></div>`: "SEO-BREADCRUMB-002",
	}
	for doc, want := range cases {
		r := extractAt(t, "https://example.com/", doc)
		if got := ids(r.BreadcrumbFindings("u")); got != want {
			t.Errorf("%s\nfindings = %q, want %q", doc, got, want)
		}
	}
}

func TestMicrodataValues(t *testing.T) {
	r := extractAt(t, "https://example.com/dir/", `<div itemscope itemtype="https://schema.org/Product">
<span itemprop="name brand"> Archiwum   X </span>
<img itemprop="image" src="a.jpg">
<time itemprop="releaseDate" datetime="2026-01-01">1 stycznia</time>
<data itemprop="sku" value="123">SKU</data>
<link itemprop="availability" href="https://schema.org/InStock">
<div itemprop="offers" itemscope itemtype="https://schema.org/Offer"><meta itemprop="price" content="99"></div>
<template><span itemprop="name">hidden</span></template>
</div><span itemprop="orphan">no item</span>`)
	if len(r.Microdata) != 2 {
		t.Fatalf("items = %+v", r.Microdata)
	}
	p := r.Microdata[0].Props
	checks := map[string]string{
		"name": "Archiwum X", "brand": "Archiwum X", "image": "https://example.com/dir/a.jpg",
		"releaseDate": "2026-01-01", "sku": "123", "availability": "https://schema.org/InStock",
	}
	for k, want := range checks {
		if len(p[k]) != 1 || p[k][0].String() != want {
			t.Errorf("%s = %+v, want %q", k, p[k], want)
		}
	}
	if len(p["offers"]) != 1 || p["offers"][0].Item == nil || p["offers"][0].Item.Props["price"][0].Text != "99" {
		t.Errorf("offers = %+v", p["offers"])
	}
	if got := strings.Join(r.Types(), ","); got != "Offer,Product" {
		t.Errorf("types = %s", got)
	}
}

func TestMicrodataLimit(t *testing.T) {
	doc := strings.Repeat(`<div itemscope itemtype="https://schema.org/Thing"><span itemprop="name">x</span></div>`, MaxMicroItems+50)
	r := extractAt(t, "https://example.com/", doc)
	if len(r.Microdata) != MaxMicroItems {
		t.Errorf("items = %d", len(r.Microdata))
	}
}
