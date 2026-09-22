package terms

import "testing"

func TestWeightedZonesAndKeywords(t *testing.T) {
	r := Weigh([]Zones{{URL: "a", Title: "archiwista", Body: "archiwum"}, {URL: "b", Body: "archiwum"}}, []string{"absent"})
	if r.Pages["a"][0].Term != "archiwista" || r.Pages["a"][0].Strength != 100 {
		t.Fatal(r)
	}
	for _, list := range r.Pages {
		for _, s := range list {
			if s.Strength < 0 || s.Strength > 100 {
				t.Fatal(s)
			}
			if s.Term == "absent" && s.Strength != 0 {
				t.Fatal(s)
			}
		}
	}
	if got := Tokenize("jest oraz nie się THE i ŻÓŁĆ"); len(got) != 1 || got[0] != "żółć" {
		t.Fatal(got)
	}
}
