package report

import (
	"io"
	"strings"
	"unicode"
)

// TerminalText removes control and directional formatting characters from
// hostile site strings and saved reports before terminal display.
func TerminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if (unicode.IsControl(r) && r != '\n' && r != '\t') || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, s)
}

type terminalWriter struct{ io.Writer }

func (w terminalWriter) Write(p []byte) (int, error) {
	_, err := io.WriteString(w.Writer, TerminalText(string(p)))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
