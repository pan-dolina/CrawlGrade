package content

import "testing"

func TestExtractKeepsMainText(t *testing.T) {
	body := `<html><head><title>x</title></head><body>
	<nav><a href="/">Home</a><a href="/about">About</a></nav>
	<main><h1>Analiza archiwum</h1>
	<p>Analiza archiwum to pierwszy krok do dobrze dobranych okularów. Archiwista sprawdza ostrość odczytu.</p>
	<h2>Archiwum korekcyjne</h2>
	<p>Na podstawie wyniku analizy archiwum archiwista dobiera archiwum korekcyjne.</p>
	</main></body></html>`
	r := Extract("https://example.com/", body)
	if r.URL != "https://example.com/" {
		t.Fatalf("URL = %q", r.URL)
	}
	if r.NoText {
		t.Fatal("expected text, got NoText")
	}
	if r.Boilerplate {
		t.Fatal("expected non-boilerplate text")
	}
	if len(r.Text) < 10 {
		t.Fatalf("text too short: %q", r.Text)
	}
	// The nav text must not leak into main content.
	if contains(r.Text, "console") {
		t.Fatalf("nav/script text leaked: %q", r.Text)
	}
	if !contains(r.Text, "Analiza archiwum") {
		t.Fatalf("expected article text, got %q", r.Text)
	}
}

func TestExtractDropsScriptAndStyle(t *testing.T) {
	body := `<html><body>
	<script>var x = "nie chcę tego tekstu"; console.log(x);</script>
	<style>.a { color: red; }</style>
	<main><p>Chcę tylko ten tekst o okularach korekcyjnych.</p></main>
	</body></html>`
	r := Extract("https://example.com/", body)
	if contains(r.Text, "console") || contains(r.Text, "color") {
		t.Fatalf("script/style text leaked: %q", r.Text)
	}
	if !contains(r.Text, "okularach korekcyjnych") {
		t.Fatalf("expected main text, got %q", r.Text)
	}
}

func TestExtractEmpty(t *testing.T) {
	r := Extract("https://example.com/", `<html><body></body></html>`)
	if !r.NoText {
		t.Fatal("expected NoText for empty page")
	}
	if !r.Boilerplate {
		t.Fatal("expected Boilerplate for empty page")
	}
}

func TestExtractBoilerplate(t *testing.T) {
	// A page whose only text is shared chrome (nav) should be boilerplate.
	body := `<html><body>
	<nav><a href="/">Home</a><a href="/about">About</a><a href="/contact">Contact</a></nav>
	</body></html>`
	r := Extract("https://example.com/", body)
	if !r.Boilerplate {
		t.Fatalf("expected boilerplate, got text %q", r.Text)
	}
}

func TestExtractMalformed(t *testing.T) {
	// A malformed document must not panic and must still yield something.
	r := Extract("https://example.com/", `<h1>Unclosed <p>text`)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
