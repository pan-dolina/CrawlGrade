package terms

import (
	"slices"
	"testing"
)

func TestGrams(t *testing.T) {
	got := Grams([]string{"a", "b", "c"})
	want := []Ngram{
		{Tokens: []string{"a"}},
		{Tokens: []string{"a", "b"}},
		{Tokens: []string{"a", "b", "c"}},
		{Tokens: []string{"b"}},
		{Tokens: []string{"b", "c"}},
		{Tokens: []string{"c"}},
	}
	if !slices.EqualFunc(got, want, func(x, y Ngram) bool { return slices.Equal(x.Tokens, y.Tokens) }) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGramsEmpty(t *testing.T) {
	if got := Grams(nil); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := Grams([]string{"single"}); len(got) != 1 {
		t.Fatalf("single token: got %v", got)
	}
}

func TestTopCounts(t *testing.T) {
	counts := map[string]int{
		"archiwum korekcyjne": 3,
		"archiwum":            3,
		"progresywne":         1,
		"okładki":             2,
	}
	top := Top(counts, 2)
	if len(top) != 2 {
		t.Fatalf("got %v", top)
	}
	// Ties broken alphabetically: "archiwum" before "archiwum korekcyjne".
	if top[0].String() != "archiwum" || top[1].String() != "archiwum korekcyjne" {
		t.Fatalf("got %v", top)
	}
}

func TestTopLimit(t *testing.T) {
	counts := map[string]int{"a": 1, "b": 1, "c": 1}
	if got := Top(counts, 2); len(got) != 2 {
		t.Fatalf("limit not applied: %v", got)
	}
}
