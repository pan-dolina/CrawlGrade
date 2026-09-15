// Package terms tokenizes page text into n-grams and measures how strongly
// those n-grams are associated with the pages they appear on.
//
// Tokenization is deliberately conservative: it keeps runs of letters and
// digits and drops everything else. Stopwords remove the most common function
// words that carry no topical signal in either Polish or English. The same
// small stopword list is used for both languages so that a mixed-language
// page is handled consistently.
package terms

import (
	"strings"
	"unicode"
)

// MaxGram is the largest n-gram CrawlGrade extracts (a trigram).
const MaxGram = 3

// MinGram is the smallest n-gram CrawlGrade extracts (a unigram).
const MinGram = 1

// MaxTokenRunes bounds a single token so a hostile document cannot produce
// gigantic n-grams. Tokens longer than this are dropped.
const MaxTokenRunes = 40

// Stopwords is the set of function words removed before measuring term
// strength. It is the union of common Polish and English stopwords. A token
// is lower-cased before the membership test, so the set holds lower-case
// forms.
var Stopwords = map[string]bool{
	// English function words.
	"a": true, "about": true, "above": true, "after": true, "again": true,
	"against": true, "all": true, "also": true, "am": true, "an": true,
	"and": true, "any": true, "are": true, "as": true, "at": true,
	"be": true, "because": true, "been": true, "before": true, "being": true,
	"below": true, "between": true, "both": true, "but": true, "by": true,
	"can": true, "cannot": true, "did": true, "do": true, "does": true,
	"doing": true, "done": true, "down": true, "during": true, "each": true,
	"else": true, "few": true, "for": true, "from": true, "further": true,
	"had": true, "has": true, "have": true, "having": true, "he": true,
	"her": true, "here": true, "hers": true, "herself": true, "him": true,
	"himself": true, "his": true, "how": true, "i": true, "if": true,
	"in": true, "into": true, "is": true, "it": true, "its": true,
	"itself": true, "just": true, "me": true, "more": true, "most": true,
	"my": true, "myself": true, "no": true, "nor": true, "not": true,
	"now": true, "of": true, "off": true, "on": true, "once": true,
	"only": true, "or": true, "other": true, "our": true, "ours": true,
	"ourselves": true, "out": true, "over": true, "own": true, "same": true,
	"she": true, "should": true, "so": true, "some": true, "such": true,
	"than": true, "that": true, "the": true, "their": true, "theirs": true,
	"they": true, "them": true, "themselves": true, "then": true,
	"there": true, "these": true, "this": true, "those": true,
	"through": true, "to": true, "too": true, "under": true, "until": true,
	"up": true, "us": true, "very": true, "was": true, "we": true,
	"were": true, "what": true, "when": true, "where": true, "which": true,
	"while": true, "who": true, "whom": true, "why": true, "will": true,
	"with": true, "would": true, "you": true, "your": true, "yours": true,
	"yourself": true, "yourselves": true,
	// Polish function words (unique; English stopwords above cover the
	// shared tokens).
	"ab": true, "ac": true, "ale": true, "ani": true, "aż": true,
	"być": true, "cały": true, "czy": true, "czyli": true, "dlaczego": true,
	"dlatego": true, "dlitego": true, "gdy": true, "gdyż": true, "iz": true,
	"ja": true, "jak": true, "jakże": true, "już": true, "każdy": true,
	"kilka": true, "ku": true, "le": true, "mach": true, "mi": true,
	"można": true, "niż": true, "ob": true, "od": true, "oraz": true,
	"oto": true, "owe": true, "owszem": true, "pod": true, "ponieważ": true,
	"przez": true, "przy": true, "reg": true, "się": true, "skąd": true,
	"także": true, "teraz": true, "u": true, "bez": true, "za": true,
	"zatym": true, "żaden": true, "żadna": true, "żadne": true,
	"żadnego": true, "żadnej": true, "żadem": true, "że": true,
}

// Tokenize splits s into lower-cased tokens, keeping runs of letters and
// digits. It drops empty tokens, stopwords and tokens longer than
// MaxTokenRunes. The result is stable regardless of input case.
func Tokenize(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		t := strings.ToLower(b.String())
		b.Reset()
		if Stopwords[t] {
			return
		}
		if len(t) > MaxTokenRunes {
			return
		}
		out = append(out, t)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
