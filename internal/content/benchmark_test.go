package content

import (
	"strings"
	"testing"
)

func BenchmarkExtract(b *testing.B) {
	body := "<main>" + strings.Repeat("<p>Analiza archiwum i archiwum korekcyjne.</p>", 100) + "</main>"
	b.ReportAllocs()
	for b.Loop() {
		Extract("https://example.com/", body)
	}
}
