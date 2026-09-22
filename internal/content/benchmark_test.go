package content

import (
	"strings"
	"testing"
)

func BenchmarkExtract(b *testing.B) {
	body := "<main>" + strings.Repeat("<p>Badanie wzroku i okulary korekcyjne.</p>", 100) + "</main>"
	b.ReportAllocs()
	for b.Loop() {
		Extract("https://example.com/", body)
	}
}
