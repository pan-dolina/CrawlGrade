package sitemap

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
)

// gzip magic bytes (RFC 1952).
var gzipMagic = []byte{0x1f, 0x8b}

// limitedBody reads a sitemap body, transparently decompressing gzip files,
// and fails once more than max bytes have been produced. Unlike
// io.LimitReader it reports the overflow instead of silently truncating, so
// a compression bomb is reported as such and not as a malformed document.
type limitedBody struct {
	r        io.Reader
	remain   int64
	exceeded bool
	gzip     bool
}

func (l *limitedBody) Read(p []byte) (int, error) {
	if l.remain <= 0 {
		// Probe one byte to distinguish "exactly at the limit" from "over".
		var one [1]byte
		n, err := l.r.Read(one[:])
		if n > 0 {
			l.exceeded = true
			return 0, fetcher.ErrTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > l.remain {
		p = p[:l.remain]
	}
	n, err := l.r.Read(p)
	l.remain -= int64(n)
	return n, err
}

// openBody returns a reader over the sitemap document in resp. HTTP
// Content-Encoding has already been removed by the fetcher (under its own
// limits); this handles files that are themselves gzip archives, such as
// sitemap.xml.gz served as application/gzip.
func openBody(resp *fetcher.Response, max int64) (*limitedBody, error) {
	raw := resp.Body
	if !isGzipFile(resp) {
		return &limitedBody{r: bytes.NewReader(raw), remain: max}, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip sitemap: %w", err)
	}
	// A single member is expected; concatenated members are still bounded
	// by the same output limit.
	return &limitedBody{r: zr, remain: max, gzip: true}, nil
}

func isGzipFile(resp *fetcher.Response) bool {
	if bytes.HasPrefix(resp.Body, gzipMagic) {
		return true
	}
	switch resp.ContentType {
	case "application/gzip", "application/x-gzip":
		return true
	}
	return strings.HasSuffix(strings.ToLower(resp.FinalURL), ".gz") && len(resp.Body) > 0 && !bytes.HasPrefix(bytes.TrimSpace(resp.Body), []byte("<"))
}
