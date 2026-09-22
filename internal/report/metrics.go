package report

import "github.com/pan-dolina/crawlgrade/internal/terms"

type PageMetric struct {
	URL       string        `json:"url"`
	Status    int           `json:"status"`
	Depth     int           `json:"depth"`
	Indexable bool          `json:"indexable"`
	Inbound   int           `json:"inbound"`
	Outbound  int           `json:"outbound"`
	Terms     []terms.Score `json:"terms,omitempty"`
}

// Scores describe observed technical findings, never search rankings.
type Scores struct {
	SEO        int `json:"seo"`
	WebHygiene int `json:"web_hygiene"`
}
