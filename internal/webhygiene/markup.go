package webhygiene

import (
	"github.com/pan-dolina/crawlgrade/internal/findings"
	"golang.org/x/net/html"
	"net/url"
	"strings"
)

// MarkupFindings observes resource references without making requests.
func MarkupFindings(pageURL string, root *html.Node) []findings.Finding {
	u, err := url.Parse(pageURL)
	if err != nil || u.Scheme != "https" {
		return nil
	}
	var mixed []string
	stack := []*html.Node{root}
	for len(stack) > 0 && len(mixed) < 20 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n.Type == html.ElementNode && n.Data == "template" {
			continue
		}
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "src" && (n.Data == "img" || n.Data == "script" || n.Data == "iframe" || n.Data == "audio" || n.Data == "video" || n.Data == "source") && strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.Val)), "http://") {
					mixed = append(mixed, a.Val)
				}
			}
		}
		for c := n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}
	if len(mixed) > 0 {
		return []findings.Finding{findings.WebHygieneMixed.New(pageURL, mixed...)}
	}
	return nil
}
