package terms

import (
	"testing"
)

func TestTokenizeDropsStopwordsAndPunctuation(t *testing.T) {
	got := Tokenize("Archiwum, korekcyjne — i archiwum progresywne!")
	want := []string{"archiwum", "korekcyjne", "archiwum", "progresywne"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestTokenizeLowercases(t *testing.T) {
	got := Tokenize("ARCHIWUM Korekcyjne")
	if len(got) != 2 || got[0] != "archiwum" || got[1] != "korekcyjne" {
		t.Fatalf("got %v", got)
	}
}

func TestTokenizeKeepsDigits(t *testing.T) {
	got := Tokenize("model 2024 abc123")
	want := []string{"model", "2024", "abc123"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestTokenizeEmpty(t *testing.T) {
	if got := Tokenize("   ,.-;: "); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := Tokenize("i"); len(got) != 0 {
		t.Fatalf("stopword-only input should be empty, got %v", got)
	}
}
