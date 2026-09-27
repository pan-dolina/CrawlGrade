package duplicates

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/terms"
)

func BenchmarkFingerprints(b *testing.B) {
	const numPages = 200
	baseText := "CrawlGrade to profesjonalny audytor techniczny SEO analizujący strukturę serwisu, metadane, nagłówki, kanoniczne adresy URL, mikrodane oraz higienę bezpieczeństwa."
	tokens := terms.Tokenize(baseText)

	pages := make([]*Page, numPages)
	for i := 0; i < numPages; i++ {
		txt := baseText
		if i%5 == 0 {
			txt = strings.Replace(txt, "profesjonalny", "zaawansowany", 1)
		}
		pages[i] = &Page{
			URL:    fmt.Sprintf("https://example.com/p/%d", i),
			Text:   txt,
			Tokens: terms.Tokenize(txt),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = simHash(tokens)
		_ = Analyze(pages)
	}
}
