package terms

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type Zones struct{ URL, Title, Description, Headings, Body string }
type Score struct {
	Term     string  `json:"term"`
	Strength float64 `json:"strength"`
}
type Weighted struct {
	Pages map[string][]Score `json:"pages"`
	Site  []Score            `json:"site"`
	Weak  []Score            `json:"weak,omitempty"`
}

// IsKeyword reports whether term is a meaningful topical keyword rather than a
// bare number or short symbol fragment.
func IsKeyword(term string) bool {
	tokens := strings.Fields(term)
	if len(tokens) == 0 {
		return false
	}
	hasLongWord := false
	for _, tok := range tokens {
		letters := 0
		runes := 0
		for _, r := range tok {
			runes++
			if unicode.IsLetter(r) {
				letters++
			}
		}
		if letters == 0 || runes < 2 {
			return false
		}
		if letters >= 3 {
			hasLongWord = true
		}
	}
	return hasLongWord
}

// Weigh applies zone weights (title 4, headings 3, description 2, body 1),
// log TF and smoothed IDF. Scores are normalized to the strongest term in
// each document; site scores use mean raw weights before normalization.
func Weigh(pages []Zones, keywords []string) Weighted {
	df := map[string]int{}
	metaZone := map[string]bool{}
	headingZone := map[string]bool{}
	counts := make([]map[string]int, len(pages))
	for i, p := range pages {
		counts[i] = map[string]int{}
		for _, z := range []struct {
			text   string
			weight int
			isMeta bool
			isHead bool
		}{
			{p.Title, 4, true, false},
			{p.Headings, 3, false, true},
			{p.Description, 2, true, false},
			{p.Body, 1, false, false},
		} {
			for term, n := range Count(Tokenize(z.text)) {
				counts[i][term] += n * z.weight
				if z.isMeta {
					metaZone[term] = true
				}
				if z.isHead {
					headingZone[term] = true
				}
			}
		}
		for term := range counts[i] {
			df[term]++
		}
	}
	res := Weighted{Pages: map[string][]Score{}}
	site := map[string]float64{}
	for i, p := range pages {
		weights := map[string]float64{}
		for term, n := range counts[i] {
			v := (1 + math.Log(float64(n))) * (1 + math.Log(float64(1+len(pages))/float64(1+df[term])))
			weights[term] = v
			site[term] += v / float64(len(pages))
		}
		res.Pages[p.URL] = topScores(weights, keywords)
	}
	res.Site = topScores(site, keywords)
	res.Weak = weakScores(site, metaZone, headingZone)
	return res
}

func weakScores(weights map[string]float64, metaZone, headingZone map[string]bool) []Score {
	maxWeight := 0.0
	for _, w := range weights {
		maxWeight = math.Max(maxWeight, w)
	}
	if maxWeight <= 0 {
		return nil
	}
	var meta, head, rest []Score
	for term, w := range weights {
		if !IsKeyword(term) {
			continue
		}
		s := Score{Term: term, Strength: math.Round(1000*w/maxWeight) / 10}
		if s.Strength <= 0 || s.Strength >= 50 {
			continue
		}
		switch {
		case metaZone[term]:
			meta = append(meta, s)
		case headingZone[term]:
			head = append(head, s)
		default:
			rest = append(rest, s)
		}
	}
	less := func(a []Score) func(i, j int) bool {
		return func(i, j int) bool {
			if a[i].Strength != a[j].Strength {
				return a[i].Strength < a[j].Strength
			}
			// Prefer shorter n-grams when tied so the base keyword is shown
			// rather than a 3-word sentence fragment with the same count.
			wi, wj := strings.Count(a[i].Term, " "), strings.Count(a[j].Term, " ")
			if wi != wj {
				return wi < wj
			}
			return a[i].Term < a[j].Term
		}
	}
	sort.Slice(meta, less(meta))
	sort.Slice(head, less(head))
	sort.Slice(rest, less(rest))
	candidates := append(append(meta, head...), rest...)
	var out []Score
	usedTokens := map[string]bool{}
	for _, s := range candidates {
		tokens := strings.Fields(s.Term)
		overlap := false
		for _, tok := range tokens {
			if usedTokens[tok] {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		for _, tok := range tokens {
			usedTokens[tok] = true
		}
		out = append(out, s)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

func topScores(weights map[string]float64, keywords []string) []Score {
	maxWeight := 0.0
	for _, w := range weights {
		maxWeight = math.Max(maxWeight, w)
	}
	var out []Score
	for term, w := range weights {
		if maxWeight > 0 {
			out = append(out, Score{term, math.Round(1000*w/maxWeight) / 10})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Strength != out[j].Strength {
			return out[i].Strength > out[j].Strength
		}
		return out[i].Term < out[j].Term
	})
	if len(out) > 20 {
		out = out[:20]
	}
	seen := map[string]bool{}
	for _, s := range out {
		seen[s.Term] = true
	}
	for _, raw := range keywords {
		term := strings.Join(Tokenize(raw), " ")
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		v := 0.0
		if maxWeight > 0 {
			v = math.Round(1000*weights[term]/maxWeight) / 10
		}
		out = append(out, Score{term, v})
		if len(out) >= 120 {
			break
		}
	}
	return out
}
