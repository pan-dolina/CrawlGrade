// Package urlnorm normalizes URLs, decides crawl scope and detects URL
// patterns that indicate crawl traps.
//
// Normalization is conservative: it only applies transformations that do not
// change which resource a URL identifies on a conforming server (RFC 3986
// section 6.2.2 and 6.2.3). Paths keep their case and trailing slashes, and
// query parameters keep their order.
package urlnorm

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// MaxURLLength is the longest URL CrawlGrade accepts.
const MaxURLLength = 2048

// Errors returned by Parse and Resolve.
var (
	ErrNotHTTP     = errors.New("not an http or https URL")
	ErrInvalidURL  = errors.New("invalid URL")
	ErrCredentials = errors.New("URL contains credentials")
	ErrTooLong     = errors.New("URL too long")
)

var idnaProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule(),
	idna.ValidateLabels(true),
	idna.StrictDomainName(false), // allow underscores used by some hosts
	idna.Transitional(false),
)

// ParseStart parses a URL given by the user. A missing scheme defaults to
// https.
func ParseStart(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidURL)
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	return Parse(raw)
}

// Parse parses and normalizes an absolute URL.
func Parse(raw string) (*url.URL, error) {
	return Resolve(nil, raw)
}

// Resolve resolves ref against base (which may be nil for absolute refs)
// the way a browser resolves an href attribute, and normalizes the result.
func Resolve(base *url.URL, ref string) (*url.URL, error) {
	ref = cleanHref(ref)
	if len(ref) > MaxURLLength {
		return nil, ErrTooLong
	}
	u, err := url.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	return Normalize(u)
}

// cleanHref applies the URL parser's input preprocessing: leading and
// trailing C0 controls and spaces are stripped, and ASCII tab and newline
// characters are removed.
func cleanHref(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool { return r <= 0x20 })
	if strings.ContainsAny(s, "\t\n\r") {
		s = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(s)
	}
	return s
}

// Normalize returns a normalized copy of an absolute URL.
func Normalize(in *url.URL) (*url.URL, error) {
	scheme := strings.ToLower(in.Scheme)
	if scheme != "http" && scheme != "https" {
		if scheme == "" {
			return nil, fmt.Errorf("%w: relative URL", ErrInvalidURL)
		}
		return nil, fmt.Errorf("%w: %s", ErrNotHTTP, scheme)
	}
	if in.User != nil {
		return nil, ErrCredentials
	}
	if in.Opaque != "" {
		return nil, fmt.Errorf("%w: opaque URL", ErrInvalidURL)
	}
	host, err := normalizeHost(in.Hostname())
	if err != nil {
		return nil, err
	}
	port := in.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		for _, c := range port {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("%w: port %q", ErrInvalidURL, port)
			}
		}
		port = strings.TrimLeft(port, "0")
		if port == "" || len(port) > 5 {
			return nil, fmt.Errorf("%w: port", ErrInvalidURL)
		}
	}

	var b strings.Builder
	b.WriteString(scheme)
	b.WriteString("://")
	if strings.Contains(host, ":") {
		b.WriteString("[" + host + "]")
	} else {
		b.WriteString(host)
	}
	if port != "" {
		b.WriteString(":" + port)
	}
	path := removeDotSegments(normalizePercent(in.EscapedPath(), pathSafe))
	if path == "" {
		path = "/"
	}
	b.WriteString(path)
	if in.RawQuery != "" {
		b.WriteString("?")
		b.WriteString(normalizePercent(in.RawQuery, querySafe))
	}
	s := b.String()
	if len(s) > MaxURLLength {
		return nil, ErrTooLong
	}
	out, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	return out, nil
}

func normalizeHost(host string) (string, error) {
	if host == "" {
		return "", fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	if ip := net.ParseIP(host); ip != nil {
		if strings.Contains(host, ":") {
			return ip.String(), nil
		}
		return ip.To4().String(), nil
	}
	if strings.ContainsAny(host, "%") {
		// Percent-encoded host names are decoded by some parsers and not
		// others; refusing them avoids ambiguity about what is contacted.
		return "", fmt.Errorf("%w: percent-encoded host", ErrInvalidURL)
	}
	host = strings.TrimSuffix(host, ".")
	ascii, err := idnaProfile.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("%w: host %q: %v", ErrInvalidURL, host, err)
	}
	ascii = strings.ToLower(ascii)
	if ascii == "" || len(ascii) > 253 {
		return "", fmt.Errorf("%w: host length", ErrInvalidURL)
	}
	for _, c := range ascii {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_') {
			return "", fmt.Errorf("%w: host %q", ErrInvalidURL, ascii)
		}
	}
	return ascii, nil
}

// removeDotSegments implements RFC 3986 section 5.2.4 on an escaped path.
func removeDotSegments(p string) string {
	if p == "" || !strings.Contains(p, ".") {
		return p
	}
	segs := strings.Split(p, "/")
	out := make([]string, 0, len(segs))
	for i, s := range segs {
		switch s {
		case ".":
			if i == len(segs)-1 {
				out = append(out, "")
			}
		case "..":
			if len(out) > 1 {
				out = out[:len(out)-1]
			}
			if i == len(segs)-1 {
				out = append(out, "")
			}
		default:
			out = append(out, s)
		}
	}
	r := strings.Join(out, "/")
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(r, "/") {
		r = "/" + r
	}
	return r
}

const hexDigits = "0123456789ABCDEF"

func isUnreserved(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// pathSafe and querySafe list bytes that may appear literally.
func pathSafe(c byte) bool {
	return isUnreserved(c) || strings.IndexByte("!$&'()*+,;=:@/", c) >= 0
}

func querySafe(c byte) bool {
	return pathSafe(c) || c == '?'
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// normalizePercent uppercases percent-encoding hex digits, decodes encoded
// unreserved characters and encodes bytes that may not appear literally.
func normalizePercent(s string, safe func(byte) bool) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' {
			if i+2 < len(s) {
				hi, ok1 := unhex(s[i+1])
				lo, ok2 := unhex(s[i+2])
				if ok1 && ok2 {
					v := hi<<4 | lo
					if isUnreserved(v) {
						b.WriteByte(v)
					} else {
						b.WriteByte('%')
						b.WriteByte(hexDigits[hi])
						b.WriteByte(hexDigits[lo])
					}
					i += 2
					continue
				}
			}
			b.WriteString("%25")
			continue
		}
		if safe(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigits[c>>4])
		b.WriteByte(hexDigits[c&0x0F])
	}
	return b.String()
}
