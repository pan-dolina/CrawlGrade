package netguard

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
)

type staticResolver map[string][]netip.Addr

func (r staticResolver) LookupNetIP(_ context.Context, network, host string) ([]netip.Addr, error) {
	ips, ok := r[network+"/"+host]
	if !ok {
		ips, ok = r[host]
	}
	if !ok {
		return nil, &net.DNSError{Err: "not found", Name: host, IsNotFound: true}
	}
	return ips, nil
}

func TestControlHookBlocksSocketAddress(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	d := &Dialer{}
	// Bypass DialContext's own checks to prove the Control hook alone stops
	// the connection.
	_, err = d.netDialer().DialContext(context.Background(), "tcp", ln.Addr().String())
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("Control hook did not block loopback: %v", err)
	}
}

func TestDialContextValidation(t *testing.T) {
	d := &Dialer{Resolver: staticResolver{"empty.test": {}, "ip6/v6.test": {netip.MustParseAddr("::1")}}}
	ctx := context.Background()
	if _, err := d.DialContext(ctx, "tcp", "no-port"); err == nil {
		t.Error("address without port accepted")
	}
	if _, err := d.DialContext(ctx, "tcp", "example.test:0"); err == nil {
		t.Error("port 0 accepted")
	}
	if _, err := d.DialContext(ctx, "tcp", "example.test:99999"); err == nil {
		t.Error("port out of range accepted")
	}
	var dnsErr *net.DNSError
	if _, err := d.DialContext(ctx, "tcp", "empty.test:80"); !errors.As(err, &dnsErr) {
		t.Errorf("empty answer: %v", err)
	}
	if _, err := d.DialContext(ctx, "tcp", "missing.test:80"); !errors.As(err, &dnsErr) {
		t.Errorf("missing host: %v", err)
	}
	if _, err := d.DialContext(ctx, "tcp6", "v6.test:80"); !errors.Is(err, ErrBlocked) {
		t.Errorf("tcp6 lookup: %v", err)
	}
}

func TestDialContextTriesCheckedAddressesInOrder(t *testing.T) {
	var dialed []string
	d := &Dialer{
		Resolver: staticResolver{"multi.test": {netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("2606:2800:220:1::1")}},
		Dial: func(_ context.Context, _, addr string) (net.Conn, error) {
			dialed = append(dialed, addr)
			return nil, errors.New("unreachable")
		},
	}
	_, err := d.DialContext(context.Background(), "tcp", "multi.test:443")
	if err == nil || len(dialed) != 2 || dialed[0] != "93.184.216.34:443" || dialed[1] != "[2606:2800:220:1::1]:443" {
		t.Fatalf("dialed=%v err=%v", dialed, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	dialed = nil
	d.Dial = func(context.Context, string, string) (net.Conn, error) {
		dialed = append(dialed, "x")
		cancel()
		return nil, context.Canceled
	}
	if _, err := d.DialContext(ctx, "tcp", "multi.test:443"); !errors.Is(err, context.Canceled) || len(dialed) != 1 {
		t.Errorf("cancelled dial continued: dialed=%v err=%v", dialed, err)
	}
}
