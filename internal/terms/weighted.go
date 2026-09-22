package terms

import (
	"math"
	"sort"
	"strings"
)

type Zones struct{ URL, Title, Description, Headings, Body string }
type Score struct {
	Term     string  `json:"term"`
	Strength float64 `json:"strength"`
}
type Weighted struct {
	Pages map[string][]Score `json:"pages"`
	Site  []Score            `json:"site"`
}

// Weigh applies zone weights (title 4, headings 3, description 2, body 1),
// log TF and smoothed IDF. Scores are normalized to the strongest term in
// each document; site scores use mean raw weights before normalization.
func Weigh(pages []Zones, keywords []string) Weighted {
	df := map[string]int{}
	counts := make([]map[string]int, len(pages))
	for i, p := range pages {
		counts[i] = map[string]int{}
		for _, z := range []struct {
			text   string
			weight int
		}{{p.Title, 4}, {p.Headings, 3}, {p.Description, 2}, {p.Body, 1}} {
			for term, n := range Count(Tokenize(z.text)) {
				counts[i][term] += n * z.weight
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
	return res
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
