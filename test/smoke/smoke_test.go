//go:build smoke

package smoke

import (
	"encoding/json"
	"github.com/pan-dolina/crawlgrade/internal/testsite"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestReleaseBinary(t *testing.T) {
	bin := os.Getenv("CRAWLGRADE_BIN")
	if bin == "" {
		t.Fatal("CRAWLGRADE_BIN is required")
	}
	out, err := exec.Command(bin, "version", "--json").Output()
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatal(err)
	}
	if want := os.Getenv("CRAWLGRADE_EXPECT_VERSION"); want != "" && v["version"] != want {
		t.Fatalf("version: %v", v)
	}
	srv := httptest.NewServer(testsite.Handler())
	defer srv.Close()
	out, err = exec.Command(bin, srv.URL, "--allow-private", "--json", "--max-depth", "0", "--requests-per-second", "10000").Output()
	if err != nil || !json.Valid(out) {
		t.Fatalf("audit: %v %s", err, out)
	}
}
