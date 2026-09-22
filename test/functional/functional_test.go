package functional

import (
	"bytes"
	"encoding/json"
	"github.com/pan-dolina/crawlgrade/internal/report"
	"github.com/pan-dolina/crawlgrade/internal/testsite"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestBinary(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "crawlgrade")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "../../cmd/crawlgrade")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	srv := httptest.NewServer(testsite.Handler())
	defer srv.Close()
	run := func(want int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("%v: exit %d want %d: %s", args, code, want, stderr.String())
		}
		return out
	}
	for _, args := range [][]string{{"https://example.invalid", "--format", "bad"}, {"file:///etc/passwd"}, {"https://example.invalid", "--max-pages", "-1"}, {"https://example.invalid", "extra"}, {"https://example.invalid", "--fail-on", "bad"}, {"https://example.invalid", "--diff"}} {
		run(2, args...)
	}
	run(3, srv.URL, "--json", "--requests-per-second", "10000")
	args := []string{srv.URL, "--allow-private", "--json", "--max-pages", "60", "--max-depth", "2", "--requests-per-second", "10000"}
	a, b := run(0, args...), run(0, args...)
	normalize := func(data []byte) *report.Report {
		r, err := report.Load(data)
		if err != nil {
			t.Fatal(err)
		}
		r.Created = time.Time{}
		return r
	}
	x, _ := json.Marshal(normalize(a))
	y, _ := json.Marshal(normalize(b))
	if !bytes.Equal(x, y) {
		t.Fatal("reports differ for a stable site")
	}
	baseline := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(baseline, a, 0600); err != nil {
		t.Fatal(err)
	}
	if got := string(run(0, "diff", baseline, baseline)); got != "No changes since the baseline.\n" {
		t.Fatal(got)
	}
	html := run(0, srv.URL+"/xss", "--allow-private", "--format", "html", "--max-depth", "0", "--requests-per-second", "10000")
	if bytes.Contains(html, []byte("<script>")) || !bytes.Contains(html, []byte("Content-Security-Policy")) {
		t.Fatal("unsafe HTML report")
	}
	run(1, srv.URL, "--allow-private", "--max-depth", "0", "--fail-on", "info", "--requests-per-second", "10000")
	run(4, srv.URL+"/missing", "--allow-private", "--requests-per-second", "10000")
	// Golden records the discovered rule catalog, counts and crawl outcome.
	r := normalize(a)
	seen := map[string]bool{}
	for _, fs := range r.Groups {
		for _, f := range fs {
			seen[f.ID] = true
		}
	}
	var ids []string
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	r.Summary.StartURL = strings.ReplaceAll(r.Summary.StartURL, srv.URL, "SITE")
	projection, err := json.MarshalIndent(struct {
		Summary report.Summary
		Rules   []string
	}{r.Summary, ids}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	projection = append(projection, '\n')
	golden := filepath.Join("testdata", "audit.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, projection, 0600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(projection, want) {
		t.Fatalf("golden mismatch: %s", projection)
	}
}
