package structureddata

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	for _, seed := range []string{"", "https://example.com/a/../b?q=x", "<html><main>Zażółć gęślą jaźń</main></html>", "User-agent: *\nDisallow: /private/", "<urlset><url><loc>https://example.com/</loc></url></urlset>", `<script type="application/ld+json">{"@context":"https://schema.org","@type":"Product"}</script>`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		root, err := html.Parse(strings.NewReader(string(data)))
		if err == nil {
			r := Extract(root, nil)
			r.Findings("https://example.com/")
			r.BreadcrumbFindings("https://example.com/")
		}
	})
}
