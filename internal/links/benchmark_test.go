package links

import (
	"fmt"
	"testing"
)

func BenchmarkLinks10k(b *testing.B) {
	const numPages = 500
	const linksPerPage = 20 // 500 * 20 = 10,000 links
	pages := make([]*Page, numPages)
	for i := 0; i < numPages; i++ {
		links := make([]Link, linksPerPage)
		for j := 0; j < linksPerPage; j++ {
			target := (i + j + 1) % numPages
			links[j] = Link{
				URL:        fmt.Sprintf("https://example.com/page/%d", target),
				AnchorText: fmt.Sprintf("anchor %d", target),
			}
		}
		pages[i] = &Page{
			URL:       fmt.Sprintf("https://example.com/page/%d", i),
			Links:     links,
			Indexable: true,
			Status:    200,
			Start:     i == 0,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = Analyze(pages)
	}
}
