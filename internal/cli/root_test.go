package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errb, Getenv: func(string) string { return "" }}
	code = app.Execute(context.Background(), args)
	return out.String(), errb.String(), code
}

func TestHelpListsExitCodes(t *testing.T) {
	out, _, code := run(t, "--help")
	if code != ExitOK || !strings.Contains(out, "Exit codes:") {
		t.Fatalf("code %d, output:\n%s", code, out)
	}
	if !strings.Contains(out, "does not predict search engine rankings") {
		t.Error("help must state that rankings are not predicted")
	}
}

func TestNoArgsShowsHelp(t *testing.T) {
	out, _, code := run(t)
	if code != ExitOK || !strings.Contains(out, "Usage:") {
		t.Fatalf("code %d, output:\n%s", code, out)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	_, errOut, code := run(t, "--definitely-not-a-flag")
	if code != ExitUsage || !strings.Contains(errOut, "crawlgrade --help") {
		t.Fatalf("code %d, stderr:\n%s", code, errOut)
	}
}

func TestVersion(t *testing.T) {
	out, _, code := run(t, "version")
	if code != ExitOK || !strings.HasPrefix(out, "crawlgrade ") {
		t.Fatalf("code %d, output %q", code, out)
	}
	out, _, code = run(t, "version", "--json")
	var v map[string]any
	if code != ExitOK || json.Unmarshal([]byte(out), &v) != nil || v["version"] == "" {
		t.Fatalf("code %d, output %q", code, out)
	}
	if _, _, code := run(t, "version", "extra"); code != ExitUsage {
		t.Errorf("extra argument: code %d", code)
	}
}

func TestExitCodeMapping(t *testing.T) {
	if ExitCode(nil) != ExitOK {
		t.Error("nil error")
	}
	if ExitCode(silentExit(ExitFindings)) != ExitFindings {
		t.Error("silent exit")
	}
	if ExitCode(withCode(ExitBlocked, context.Canceled)) != ExitBlocked {
		t.Error("wrapped code")
	}
	if ExitCode(context.Canceled) != ExitUsage {
		t.Error("untyped errors are usage errors")
	}
	if silentExit(3).Error() != "exit status 3" {
		t.Error("silent exit message")
	}
}
