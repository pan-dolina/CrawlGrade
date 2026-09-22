package webhygiene

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestMixedResources(t *testing.T) {
	root, err := html.Parse(strings.NewReader(`<a href="http://example.com/">link</a><img src="http://example.com/a.png"><template><img src="http://example.com/b.png"></template>`))
	if err != nil {
		t.Fatal(err)
	}
	fs := MarkupFindings("https://example.com/", root)
	if len(fs) != 1 || len(fs[0].Evidence) != 1 {
		t.Fatal(fs)
	}
	if len(MarkupFindings("http://example.com/", root)) != 0 {
		t.Fatal("plain HTTP is not mixed content")
	}
}
