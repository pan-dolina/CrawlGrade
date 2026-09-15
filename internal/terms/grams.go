package terms

import (
	"slices"
	"strings"
)

// Ngram is a sequence of MinGram..MaxGram consecutive tokens.
type Ngram struct {
	Tokens []string
}

// String returns the n-gram as a space-joined string.
func (n Ngram) String() string {
	return strings.Join(n.Tokens, " ")
}

// Grams returns every n-gram (unigram through MaxGram) that can be built from
// tokens in order. Empty and single-token inputs yield the corresponding
// small set. The result is bounded by MaxGram * len(tokens).
func Grams(tokens []string) []Ngram {
	n := len(tokens)
	if n == 0 {
		return nil
	}
	var out []Ngram
	for start := 0; start < n; start++ {
		for length := MinGram; length <= MaxGram && start+length <= n; length++ {
			gram := Ngram{Tokens: append([]string{}, tokens[start:start+length]...)}
			out = append(out, gram)
		}
	}
	return out
}

// Count maps each distinct n-gram to the number of times it appears in tokens.
func Count(tokens []string) map[string]int {
	out := map[string]int{}
	for _, g := range Grams(tokens) {
		out[g.String()]++
	}
	return out
}

// Top returns the n-grams with the highest raw frequency, sorted by descending
// count and then alphabetically for determinism. At most limit n-grams are
// returned; limit <= 0 returns all of them.
func Top(counts map[string]int, limit int) []Ngram {
	type kv struct {
		key string
		n   int
	}
	pairs := make([]kv, 0, len(counts))
	for k, n := range counts {
		pairs = append(pairs, kv{k, n})
	}
	slices.SortFunc(pairs, func(a, b kv) int {
		if a.n != b.n {
			return b.n - a.n
		}
		return strings.Compare(a.key, b.key)
	})
	limit = min(limit, len(pairs))
	out := make([]Ngram, 0, limit)
	for _, p := range pairs[:limit] {
		out = append(out, Ngram{Tokens: strings.Split(p.key, " ")})
	}
	return out
}
