package version

import (
	"strings"
	"testing"
)

func TestGetFallsBackToDevel(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = ""
	if v := Get().Version; v == "" {
		t.Fatal("empty version")
	}
	Version = "v1.2.3"
	info := Get()
	if info.Version != "v1.2.3" || info.GoVersion == "" || !strings.Contains(info.Platform, "/") {
		t.Errorf("unexpected info: %+v", info)
	}
	if ua := UserAgent(); !strings.HasPrefix(ua, "CrawlGrade/v1.2.3 ") {
		t.Errorf("UserAgent() = %q", ua)
	}
}
