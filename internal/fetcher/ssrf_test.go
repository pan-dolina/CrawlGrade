package fetcher_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pan-dolina/crawlgrade/internal/fetcher"
	"github.com/pan-dolina/crawlgrade/internal/netguard"
)

// fakeNet simulates DNS and routes connections for "public" test addresses
// to a local httptest server, so the policy can be exercised with public
// and private addresses without touching real networks.
type fakeNet struct {
	mu      sync.Mutex
	answers map[string][][]netip.Addr // host -> successive answers
	lookups map[string]int
	dialed  []string
	route   map[string]string // checked ip:port -> real local address
}

func newFakeNet() *fakeNet {
	return &fakeNet{answers: map[string][][]netip.Addr{}, lookups: map[string]int{}, route: map[string]string{}}
}

func (f *fakeNet) set(host string, answers ...string) {
	var seq [][]netip.Addr
	for _, a := range answers {
		var ips []netip.Addr
		for _, s := range strings.Split(a, ",") {
			ips = append(ips, netip.MustParseAddr(s))
		}
		seq = append(seq, ips)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers[host] = seq
}

func (f *fakeNet) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	seq, ok := f.answers[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	i := min(f.lookups[host], len(seq)-1)
	f.lookups[host]++
	return seq[i], nil
}

func (f *fakeNet) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	f.mu.Lock()
	f.dialed = append(f.dialed, addr)
	real, ok := f.route[addr]
	f.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("fake network: no route to %s", addr)
	}
	var d net.Dialer
	return d.DialContext(ctx, network, real)
}

func (f *fakeNet) dials() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.dialed...)
}

func (f *fakeNet) client(allowPrivate bool) *fetcher.Client {
	d := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: allowPrivate}, Resolver: f, Dial: f.dial}
	return fetcher.New(fetcher.Options{DialContext: d.DialContext})
}

// publicServer starts a local server reachable as http://<host>/ where host
// resolves to a public test address.
func publicServer(t *testing.T, f *fakeNet, host, ip string, h http.Handler) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	f.set(host, ip)
	f.mu.Lock()
	f.route[netip.AddrPortFrom(netip.MustParseAddr(ip), 80).String()] = srv.Listener.Addr().String()
	f.mu.Unlock()
}

func redirectTo(loc string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	})
}

func TestPublicSiteIsReachable(t *testing.T) {
	f := newFakeNet()
	publicServer(t, f, "public.test", "93.184.216.34", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	resp, err := f.client(false).Fetch(context.Background(), fetcher.Request{URL: "http://public.test/"})
	if err != nil || string(resp.Body) != "hello" {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestRedirectToPrivateTargetsIsBlocked(t *testing.T) {
	targets := map[string]string{
		"private name":       "http://internal.test/admin",
		"metadata literal":   "http://169.254.169.254/latest/meta-data/",
		"loopback literal":   "http://127.0.0.1:6379/",
		"mapped loopback":    "http://[::ffff:127.0.0.1]/",
		"ipv6 loopback":      "http://[::1]:8080/",
		"localhost name":     "http://localhost/",
		"localhost subname":  "http://api.localhost/",
		"rfc1918 https":      "https://192.168.0.1/",
		"nat64 metadata":     "http://[64:ff9b::a9fe:a9fe]/",
		"unspecified":        "http://0.0.0.0/",
		"cgnat":              "http://100.64.0.1/",
		"link-local ipv6":    "http://[fe80::1]/",
		"unique local ipv6":  "http://[fd00::1]/",
		"trailing dot local": "http://localhost./",
	}
	for name, target := range targets {
		t.Run(name, func(t *testing.T) {
			f := newFakeNet()
			f.set("internal.test", "10.0.0.5")
			publicServer(t, f, "public.test", "93.184.216.34", redirectTo(target))
			resp, err := f.client(false).Fetch(context.Background(), fetcher.Request{URL: "http://public.test/start"})
			if !errors.Is(err, netguard.ErrBlocked) {
				t.Fatalf("err = %v", err)
			}
			if len(resp.Redirects) != 1 || resp.Redirects[0].Location != target {
				t.Errorf("redirect not recorded: %+v", resp.Redirects)
			}
			for _, d := range f.dials() {
				if d != "93.184.216.34:80" {
					t.Errorf("dialed unchecked destination %s", d)
				}
			}
		})
	}
}

func TestMetadataBlockedEvenWithAllowPrivate(t *testing.T) {
	f := newFakeNet()
	f.set("metadata.google.internal", "169.254.169.254")
	for _, target := range []string{"http://169.254.169.254/", "http://metadata.google.internal/computeMetadata/v1/", "http://[fd00:ec2::254]/"} {
		publicServer(t, f, "public.test", "93.184.216.34", redirectTo(target))
		_, err := f.client(true).Fetch(context.Background(), fetcher.Request{URL: "http://public.test/"})
		var be *netguard.BlockedError
		if !errors.As(err, &be) || be.Class != netguard.ClassMetadata {
			t.Errorf("%s: err = %v", target, err)
		}
	}
}

func TestDNSRebindingIsBlocked(t *testing.T) {
	f := newFakeNet()
	// First resolution answers with a public address, later ones with
	// loopback, as a rebinding attacker's DNS server would.
	publicServer(t, f, "rebind.test", "93.184.216.34", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		_, _ = w.Write([]byte("ok"))
	}))
	f.set("rebind.test", "93.184.216.34", "127.0.0.1")
	c := f.client(false)
	if _, err := c.Fetch(context.Background(), fetcher.Request{URL: "http://rebind.test/"}); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	c.Close()
	_, err := c.Fetch(context.Background(), fetcher.Request{URL: "http://rebind.test/"})
	if !errors.Is(err, netguard.ErrBlocked) {
		t.Fatalf("second fetch after rebinding: %v", err)
	}
	for _, d := range f.dials() {
		if strings.HasPrefix(d, "127.") {
			t.Errorf("connected to rebound address %s", d)
		}
	}
}

func TestMixedDNSAnswerIsBlocked(t *testing.T) {
	f := newFakeNet()
	f.set("mixed.test", "93.184.216.34,10.1.1.1")
	_, err := f.client(false).Fetch(context.Background(), fetcher.Request{URL: "http://mixed.test/"})
	if !errors.Is(err, netguard.ErrBlocked) {
		t.Fatalf("err = %v", err)
	}
	if d := f.dials(); len(d) != 0 {
		t.Errorf("dialed %v before rejecting the host", d)
	}
}

func TestIDNHostIsResolvedAsASCII(t *testing.T) {
	f := newFakeNet()
	publicServer(t, f, "xn--bcher-kva.test", "93.184.216.34", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Host))
	}))
	resp, err := f.client(false).Fetch(context.Background(), fetcher.Request{URL: "http://bücher.test/"})
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Body) != "xn--bcher-kva.test" {
		t.Errorf("Host header = %q", resp.Body)
	}
}

func TestRealLoopbackRequiresAllowPrivate(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer srv.Close()

	strict := &netguard.Dialer{}
	_, err := fetcher.New(fetcher.Options{DialContext: strict.DialContext}).Fetch(context.Background(), fetcher.Request{URL: srv.URL})
	if !errors.Is(err, netguard.ErrBlocked) || !strings.Contains(err.Error(), "--allow-private") {
		t.Fatalf("strict: %v", err)
	}
	// Numeric host forms that system resolvers may interpret as 127.0.0.1.
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	for _, host := range []string{"2130706433", "0x7f000001", "017700000001", "127.1"} {
		_, err := fetcher.New(fetcher.Options{DialContext: strict.DialContext}).Fetch(context.Background(),
			fetcher.Request{URL: fmt.Sprintf("http://%s:%d/", host, port)})
		if err == nil {
			t.Errorf("%s: request succeeded", host)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("strict policy let %d requests reach loopback", hits.Load())
	}

	local := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: true}}
	if _, err := fetcher.New(fetcher.Options{DialContext: local.DialContext}).Fetch(context.Background(), fetcher.Request{URL: srv.URL}); err != nil {
		t.Fatalf("allow-private: %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d", hits.Load())
	}
}

func TestLocalRedirectToMetadataWithAllowPrivate(t *testing.T) {
	srv := httptest.NewServer(redirectTo("http://169.254.169.254/latest/meta-data/iam/"))
	defer srv.Close()
	local := &netguard.Dialer{Policy: netguard.Policy{AllowPrivate: true}}
	resp, err := fetcher.New(fetcher.Options{DialContext: local.DialContext}).Fetch(context.Background(), fetcher.Request{URL: srv.URL})
	if !errors.Is(err, netguard.ErrBlocked) || resp.Status != http.StatusFound {
		t.Fatalf("status=%d err=%v", resp.Status, err)
	}
}
