// Package duplicates detects pages whose extracted main content is identical
// (exact duplicates) or very similar (near-duplicates).
//
// Exact duplicates are found by hashing the extracted text. Near-duplicates
// are found with a SimHash: a 64-bit fingerprint of the token stream that
// stays small when two documents share many tokens. Two pages are
// near-duplicates when their fingerprints are closer than a threshold, which
// is robust to the word reordering and small edits that separate real
// duplicates.
//
// Both analyses are pure functions of the extracted text, so they need no
// network and are deterministic.
package duplicates

import (
	"hash/fnv"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/content"
	"github.com/pan-dolina/crawlgrade/internal/terms"
)

// NearDuplicateThreshold is the maximum SimHash distance at which two pages
// are considered near-duplicates. A distance of 0 means identical hashes;
// larger values tolerate more divergence. The value is chosen so that pages
// differing only in a few words score low, while unrelated pages score high.
const NearDuplicateThreshold = 5

// MaxSimHashBits is the width of the SimHash fingerprint.
const MaxSimHashBits = 64

// Page is one crawled page's extracted content.
type Page struct {
	URL    string
	Text   string   // extracted main text
	Tokens []string // stopword-filtered tokens
}

// NewPage builds a Page from a content extraction result.
func NewPage(r *content.Result) *Page {
	return &Page{URL: r.URL, Text: r.Text, Tokens: terms.Tokenize(r.Text)}
}

// Result reports the duplicate groups found across pages.
type Result struct {
	// Exact lists pages that share identical extracted text. Each entry
	// names one representative URL and the URLs it duplicates.
	Exact []DuplicateGroup
	// Near lists pages whose content is near-duplicate. Each entry names one
	// representative URL and the URLs it is near-duplicate with.
	Near []DuplicateGroup
}

// DuplicateGroup names a representative page and the other pages that are
// (exact or near) duplicates of it.
type DuplicateGroup struct {
	Representative string
	Duplicates     []string
}

// Analyze compares every pair of pages and returns the duplicate groups.
func Analyze(pages []*Page) *Result {
	r := &Result{}
	fingerprints := make([]int64, len(pages))
	for i, p := range pages {
		fingerprints[i] = simHash(p.Tokens)
	}
	for i := 0; i < len(pages); i++ {
		for j := i + 1; j < len(pages); j++ {
			if pages[j] == nil || pages[i] == nil {
				continue
			}
			switch {
			case pages[i].Text == pages[j].Text && pages[i].Text != "":
				r.addExact(pages[i].URL, pages[j].URL)
			case hamming(fingerprints[i], fingerprints[j]) <= NearDuplicateThreshold:
				r.addNear(pages[i].URL, pages[j].URL)
			}
		}
	}
	return r
}

// addExact records that a duplicates b.
func (r *Result) addExact(a, b string) {
	found := false
	for _, g := range r.Exact {
		if g.Representative == a {
			g.Duplicates = append(g.Duplicates, b)
			r.Exact = replaceGroup(r.Exact, g)
			found = true
			break
		}
	}
	if !found {
		r.Exact = append(r.Exact, DuplicateGroup{Representative: a, Duplicates: []string{b}})
	}
}

// addNear records that a is near-duplicate with b.
func (r *Result) addNear(a, b string) {
	found := false
	for _, g := range r.Near {
		if g.Representative == a {
			g.Duplicates = append(g.Duplicates, b)
			r.Near = replaceGroup(r.Near, g)
			found = true
			break
		}
	}
	if !found {
		r.Near = append(r.Near, DuplicateGroup{Representative: a, Duplicates: []string{b}})
	}
}

// replaceGroup returns groups with the group matching g replaced.
func replaceGroup(groups []DuplicateGroup, g DuplicateGroup) []DuplicateGroup {
	out := make([]DuplicateGroup, 0, len(groups))
	for _, x := range groups {
		if x.Representative == g.Representative {
			out = append(out, g)
		} else {
			out = append(out, x)
		}
	}
	return out
}

// simHash computes a SimHash over the tokens. Each token contributes a
// per-bit signed vote based on its hash; the sign of the running sum at each
// bit determines the fingerprint bit.
func simHash(tokens []string) int64 {
	var sums [MaxSimHashBits]int64
	for _, tok := range tokens {
		h := fnvHash(tok)
		for i := 0; i < MaxSimHashBits; i++ {
			if h&(1<<uint(i&63)) != 0 {
				sums[i]++
			} else {
				sums[i]--
			}
		}
	}
	var hash int64
	for i := 0; i < MaxSimHashBits; i++ {
		if sums[i] < 0 {
			hash |= 1 << uint(i)
		}
	}
	return hash
}

// fnvHash returns a 64-bit FNV-1a hash of s.
func fnvHash(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// hamming counts the differing bits of a and b.
func hamming(a, b int64) int {
	x := a ^ b
	n := 0
	for x != 0 {
		x &= x - 1
		n++
	}
	return n
}

// text returns the joined duplicate URLs for evidence.
func (g DuplicateGroup) text() string {
	return strings.Join(g.Duplicates, ", ")
}
