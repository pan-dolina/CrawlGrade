package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ErrBlocked matches every error caused by the network policy.
var ErrBlocked = errors.New("destination blocked by network policy")

// BlockedError describes a rejected destination.
type BlockedError struct {
	Host  string
	IP    netip.Addr // invalid when the host name itself was rejected
	Class Class
	// AllowPrivate is true if the policy already permitted private networks.
	AllowPrivate bool
}

func (e *BlockedError) Error() string {
	var b strings.Builder
	b.WriteString("destination blocked by network policy: ")
	if e.IP.IsValid() {
		if e.Host != "" && e.Host != e.IP.String() {
			fmt.Fprintf(&b, "%s resolves to %s, a %s address", e.Host, e.IP, e.Class)
		} else {
			fmt.Fprintf(&b, "%s is a %s address", e.IP, e.Class)
		}
	} else {
		fmt.Fprintf(&b, "host %q is a %s name", e.Host, e.Class)
	}
	if !e.AllowPrivate && e.Class.AllowedWithPrivate() {
		b.WriteString(" (use --allow-private to audit local targets)")
	}
	return b.String()
}

// Unwrap makes BlockedError match ErrBlocked.
func (e *BlockedError) Unwrap() error { return ErrBlocked }

// Policy decides which destinations are allowed.
type Policy struct {
	// AllowPrivate permits loopback, private, link-local and carrier-grade
	// NAT addresses. Metadata, unspecified, multicast, broadcast and reserved
	// addresses are always blocked.
	AllowPrivate bool
}

// CheckIP returns a *BlockedError if ip is not an allowed destination.
func (p Policy) CheckIP(host string, ip netip.Addr) error {
	if ip.Zone() != "" {
		return &BlockedError{Host: host, IP: ip, Class: ClassReserved, AllowPrivate: p.AllowPrivate}
	}
	c := Classify(ip)
	if c == ClassPublic || (p.AllowPrivate && c.AllowedWithPrivate()) {
		return nil
	}
	return &BlockedError{Host: host, IP: ip, Class: c, AllowPrivate: p.AllowPrivate}
}

// CheckHost rejects host names that always refer to the local machine,
// before any resolution happens. IP literals are checked with CheckIP.
func (p Policy) CheckHost(host string) error {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if ip, err := netip.ParseAddr(host); err == nil {
		return p.CheckIP(host, ip)
	}
	if p.AllowPrivate {
		return nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return &BlockedError{Host: host, Class: ClassLoopback}
	}
	return nil
}

// Resolver looks up IP addresses; *net.Resolver implements it.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// DialFunc connects to an address.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// Dialer connects only to destinations permitted by Policy.
type Dialer struct {
	Policy   Policy
	Resolver Resolver // nil uses net.DefaultResolver
	Timeout  time.Duration
	// Dial connects to an already checked IP:port. Nil uses a net.Dialer
	// whose Control hook checks the socket address again. Tests replace it
	// to route checked addresses to local servers.
	Dial DialFunc
}

// DialContext resolves addr, checks every resolved address against the
// policy and connects to a checked IP literal. If any resolved address is
// blocked, the host is rejected: accepting only the "good" answers of a mixed
// response would let a hostile DNS server race the check.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		return nil, fmt.Errorf("invalid port in %q", addr)
	}
	if err := d.Policy.CheckHost(host); err != nil {
		return nil, err
	}

	var ips []netip.Addr
	if ip, perr := netip.ParseAddr(host); perr == nil {
		ips = []netip.Addr{ip}
	} else {
		ips, err = d.resolve(ctx, network, host)
		if err != nil {
			return nil, err
		}
	}
	for _, ip := range ips {
		if err := d.Policy.CheckIP(host, ip); err != nil {
			return nil, err
		}
	}

	dial := d.Dial
	if dial == nil {
		dial = d.netDialer().DialContext
	}
	var errs []error
	for _, ip := range ips {
		conn, err := dial(ctx, network, netip.AddrPortFrom(ip.Unmap(), uint16(port)).String())
		if err == nil {
			return conn, nil
		}
		errs = append(errs, err)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, errors.Join(errs...)
}

func (d *Dialer) resolve(ctx context.Context, network, host string) ([]netip.Addr, error) {
	r := d.Resolver
	if r == nil {
		r = net.DefaultResolver
	}
	lookupNet := "ip"
	switch network {
	case "tcp4":
		lookupNet = "ip4"
	case "tcp6":
		lookupNet = "ip6"
	}
	ips, err := r.LookupNetIP(ctx, lookupNet, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, &net.DNSError{Err: "no addresses", Name: host, IsNotFound: true}
	}
	return ips, nil
}

// netDialer returns a dialer whose Control hook re-checks the address the
// socket is about to connect to, as defence in depth.
func (d *Dialer) netDialer() *net.Dialer {
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			return d.Policy.checkSocketAddr(address)
		},
	}
}

func (p Policy) checkSocketAddr(address string) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: unexpected socket address %q", ErrBlocked, address)
	}
	return p.CheckIP("", ap.Addr())
}
