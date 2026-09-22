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
	"crypto/sha256"
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
	exact := map[[32]byte]int{}
	var hashes []int64
	for _, p := range pages {
		if p == nil || p.Text == "" || len(p.Tokens) == 0 {
			continue
		}
		digest := sha256.Sum256([]byte(p.Text))
		if i, ok := exact[digest]; ok {
			r.Exact[i].Duplicates = append(r.Exact[i].Duplicates, p.URL)
			continue
		}
		exact[digest] = len(r.Exact)
		r.Exact = append(r.Exact, DuplicateGroup{Representative: p.URL})
		hash := simHash(p.Tokens)
		matched := false
		for i, h := range hashes {
			if hamming(hash, h) <= NearDuplicateThreshold {
				r.Near[i].Duplicates = append(r.Near[i].Duplicates, p.URL)
				matched = true
				break
			}
		}
		if !matched {
			hashes = append(hashes, hash)
			r.Near = append(r.Near, DuplicateGroup{Representative: p.URL})
		}
	}
	filter := func(groups []DuplicateGroup) []DuplicateGroup {
		var out []DuplicateGroup
		for _, g := range groups {
			if len(g.Duplicates) > 0 {
				out = append(out, g)
			}
		}
		return out
	}
	r.Exact = filter(r.Exact)
	r.Near = filter(r.Near)
	return r
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
