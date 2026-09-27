package terms

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkTokenizer(b *testing.B) {
	sample := strings.Repeat("Audyt techniczny SEO i optymalizacja struktury serwisu. Comprehensive technical analysis of crawlability, metadata, structured data, internal linking, and content relevance. Słowa kluczowe i frazy kluczowe o wysokiej sile. ", 10)
	b.ReportAllocs()
	for b.Loop() {
		_ = Tokenize(sample)
	}
}

func BenchmarkNgrams(b *testing.B) {
	sample := strings.Repeat("audyt techniczny seo optymalizacja struktury serwisu crawling stron analiza linkowania ", 10)
	tokens := Tokenize(sample)
	b.ReportAllocs()
	for b.Loop() {
		_ = Grams(tokens)
	}
}

func BenchmarkTFIDF1000Docs(b *testing.B) {
	const numDocs = 1000
	pages := make([]Zones, numDocs)
	keywords := []string{"audyt seo", "technical audit", "crawlability", "metadata"}
	for i := 0; i < numDocs; i++ {
		pages[i] = Zones{
			URL:         fmt.Sprintf("https://example.com/page/%d", i),
			Title:       fmt.Sprintf("SEO Audit Service Page %d", i),
			Description: fmt.Sprintf("Comprehensive technical audit and crawl analysis for page %d", i),
			Headings:    "Crawlability and Content Relevance Analysis",
			Body:        "Our audit examines robots.txt, sitemaps, structured data, headings, duplicate content, internal linking structure, and web hygiene. Technical SEO ensures search engines can efficiently discover, parse, and index relevant pages.",
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = Weigh(pages, keywords)
	}
}
