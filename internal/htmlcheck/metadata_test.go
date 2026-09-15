package htmlcheck

import (
	"net/url"
	"strings"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/findings"
)

func parse(t *testing.T, rawURL, body string) *Page {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err := Parse([]byte(body), u, "text/html; charset=utf-8")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func ids(fs []findings.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.ID)
	}
	return strings.Join(s, ",")
}

func TestTitleAndDescription(t *testing.T) {
	good := `<title>  Salon   optyczny
	w Warszawie </title><meta name="Description" content=" Badanie wzroku, okulary korekcyjne i oprawki w centrum Warszawy. ">`
	cases := []struct {
		name, body, want string
	}{
		{"good", good, ""},
		{"missing both", `<p>hi</p>`, "SEO-TITLE-001,SEO-DESC-001"},
		{"empty", `<title> </title><meta name="description" content="">`, "SEO-TITLE-002,SEO-DESC-002"},
		{"short and long", `<title>Home</title><meta name="description" content="` + strings.Repeat("opis ", 40) + `">`, "SEO-TITLE-005,SEO-DESC-005"},
		{"long title", `<title>` + strings.Repeat("Okulary ", 10) + `</title><meta name="description" content="Badanie wzroku, okulary korekcyjne i oprawki w centrum Warszawy.">`, "SEO-TITLE-006"},
		{"multiple", good + `<title>Second title here!</title><meta name="description" content="Druga wersja opisu strony, dodana przez inny szablon.">`, "SEO-TITLE-003,SEO-DESC-003"},
		{"svg title ignored", good + `<body><svg><title>icon</title></svg></body>`, ""},
		{"title in body still counts", `<meta name="description" content="Badanie wzroku, okulary korekcyjne i oprawki w centrum Warszawy."><body><title>Late title in body</title></body>`, ""},
		{"template ignored", good + `<body><template><title>x</title><meta name="description" content="y"></template></body>`, ""},
		{"og description is not meta description", `<title>Salon optyczny w Warszawie</title><meta property="description" content="x">`, "SEO-DESC-001"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := parse(t, "https://example.com/", c.body)
			if got := ids(p.MetadataFindings()); got != c.want {
				t.Errorf("findings = %q, want %q (titles %q, descriptions %q)", got, c.want, p.Titles, p.Descriptions)
			}
		})
	}
	p := parse(t, "https://example.com/", good)
	if p.Title() != "Salon optyczny w Warszawie" {
		t.Errorf("title = %q", p.Title())
	}
}

func TestUnicodeLengthAndClipping(t *testing.T) {
	// 20 Polish characters are 20 runes but 40 bytes: not "short".
	p := parse(t, "https://example.com/", `<title>`+strings.Repeat("ż", 20)+`</title>`)
	for _, f := range p.MetadataFindings() {
		if f.ID == "SEO-TITLE-005" || f.ID == "SEO-TITLE-006" {
			t.Errorf("unexpected %s", f.ID)
		}
	}
	long := parse(t, "https://example.com/", `<title>`+strings.Repeat("ą", 5000)+`</title>`)
	if n := len([]rune(long.Title())); n != MaxTextRunes {
		t.Errorf("title clipped to %d runes", n)
	}
}

func TestLegacyCharset(t *testing.T) {
	u, _ := url.Parse("https://example.com/")
	// "Żółw" in ISO-8859-2.
	body := []byte("<html><head><title>\xAF\xF3\xB3w</title></head></html>")
	p, _, err := Parse(body, u, "text/html; charset=ISO-8859-2")
	if err != nil || p.Title() != "Żółw" {
		t.Fatalf("title = %q err = %v", p.Title(), err)
	}
	p, _, _ = Parse([]byte(`<meta charset="iso-8859-2"><title>`+"\xAF\xF3\xB3w"+`</title>`), u, "text/html")
	if p.Title() != "Żółw" {
		t.Errorf("meta charset title = %q", p.Title())
	}
}

func TestDuplicateMetadata(t *testing.T) {
	a := parse(t, "https://example.com/a", `<title>Same title</title><meta name="description" content="Same">`)
	b := parse(t, "https://example.com/b", `<title>Same  title</title><meta name="description" content="Same">`)
	c := parse(t, "https://example.com/c", `<title>Other</title><meta name="description" content="">`)
	d := parse(t, "https://example.com/d", `<title></title>`)
	e := parse(t, "https://example.com/e", `<title></title>`)
	fs := DuplicateMetadataFindings([]*Page{c, b, a, d, e})
	if got := ids(fs); got != "SEO-DESC-004,SEO-DESC-004,SEO-TITLE-004,SEO-TITLE-004" {
		// findings.Sort orders by severity first: title duplicates are medium.
		if got != "SEO-TITLE-004,SEO-TITLE-004,SEO-DESC-004,SEO-DESC-004" {
			t.Fatalf("findings = %s", got)
		}
	}
	if fs[0].URL != "https://example.com/a" || fs[0].Evidence[1] != "also on https://example.com/b" {
		t.Errorf("first finding = %+v", fs[0])
	}
}

func TestBaseHref(t *testing.T) {
	p := parse(t, "https://example.com/dir/page", `<head><base href="https://cdn.example.com/root/"><base href="/ignored/"></head>`)
	u, err := p.Resolve("x.html")
	if err != nil || u.String() != "https://cdn.example.com/root/x.html" {
		t.Errorf("resolve = %v %v", u, err)
	}
	p = parse(t, "https://example.com/dir/page", `<head><base href="javascript:alert(1)"></head>`)
	u, _ = p.Resolve("x.html")
	if u.String() != "https://example.com/dir/x.html" {
		t.Errorf("unsafe base applied: %v", u)
	}
}

func TestDeeplyNestedDocument(t *testing.T) {
	u, _ := url.Parse("https://example.com/")
	// golang.org/x/net/html refuses documents whose open element stack
	// exceeds 512 nodes. The error must surface instead of a partial page.
	_, _, err := Parse([]byte(strings.Repeat("<div>", 100000)+"<title>Deep</title>"), u, "text/html")
	if err == nil {
		t.Fatal("deeply nested document parsed without error")
	}
	// Nesting below the limit is fine and walked without recursion.
	p := parse(t, "https://example.com/", "<title>Nested title for testing</title>"+strings.Repeat("<span>", 400)+"text")
	if p.Title() != "Nested title for testing" {
		t.Errorf("title = %q", p.Title())
	}
}
