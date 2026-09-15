package netguard

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := map[string]Class{
		"8.8.8.8":                ClassPublic,
		"93.184.216.34":          ClassPublic,
		"127.0.0.1":              ClassLoopback,
		"127.255.255.254":        ClassLoopback,
		"10.1.2.3":               ClassPrivate,
		"172.16.0.1":             ClassPrivate,
		"172.31.255.255":         ClassPrivate,
		"172.32.0.1":             ClassPublic,
		"192.168.1.1":            ClassPrivate,
		"169.254.1.1":            ClassLinkLocal,
		"169.254.169.254":        ClassMetadata,
		"169.254.170.2":          ClassMetadata,
		"100.100.100.200":        ClassMetadata,
		"100.64.0.1":             ClassSharedAddress,
		"0.0.0.0":                ClassUnspecified,
		"0.1.2.3":                ClassUnspecified,
		"224.0.0.1":              ClassMulticast,
		"255.255.255.255":        ClassBroadcast,
		"240.0.0.1":              ClassReserved,
		"192.0.2.10":             ClassReserved,
		"198.18.0.1":             ClassReserved,
		"198.51.100.1":           ClassReserved,
		"203.0.113.1":            ClassReserved,
		"192.0.0.8":              ClassReserved,
		"::":                     ClassUnspecified,
		"::1":                    ClassLoopback,
		"::ffff:127.0.0.1":       ClassLoopback,
		"::ffff:169.254.169.254": ClassMetadata,
		"::ffff:8.8.8.8":         ClassPublic,
		"::127.0.0.1":            ClassReserved,
		"64:ff9b::10.0.0.1":      ClassPrivate,
		"64:ff9b::8.8.8.8":       ClassPublic,
		"64:ff9b::a9fe:a9fe":     ClassMetadata,
		"64:ff9b:1::1":           ClassPrivate,
		"2002:7f00:1::":          ClassLoopback,
		"2002:0808:0808::1":      ClassPublic,
		"2001::1":                ClassReserved,
		"2001:db8::1":            ClassReserved,
		"3fff::1":                ClassReserved,
		"fc00::1":                ClassPrivate,
		"fd00:ec2::254":          ClassMetadata,
		"fe80::1":                ClassLinkLocal,
		"fec0::1":                ClassPrivate,
		"ff02::1":                ClassMulticast,
		"100::1":                 ClassReserved,
		"4000::1":                ClassReserved,
		"2606:4700:4700::1111":   ClassPublic,
	}
	for s, want := range cases {
		if got := Classify(netip.MustParseAddr(s)); got != want {
			t.Errorf("Classify(%s) = %v, want %v", s, got, want)
		}
	}
	if Classify(netip.Addr{}) != ClassReserved {
		t.Error("invalid address must not be public")
	}
}

func TestPolicyCheckIP(t *testing.T) {
	strict, local := Policy{}, Policy{AllowPrivate: true}
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "169.254.10.10", "100.64.1.1", "::1", "fd12::1"} {
		ip := netip.MustParseAddr(s)
		if err := strict.CheckIP("", ip); !errors.Is(err, ErrBlocked) {
			t.Errorf("strict allowed %s", s)
		}
		if err := local.CheckIP("", ip); err != nil {
			t.Errorf("--allow-private blocked %s: %v", s, err)
		}
	}
	for _, s := range []string{"169.254.169.254", "fd00:ec2::254", "0.0.0.0", "224.0.0.1", "255.255.255.255", "192.0.2.1", "::ffff:169.254.169.254"} {
		if err := local.CheckIP("", netip.MustParseAddr(s)); !errors.Is(err, ErrBlocked) {
			t.Errorf("--allow-private allowed %s", s)
		}
	}
	if err := local.CheckIP("", netip.MustParseAddr("fe80::1%en0")); err == nil {
		t.Error("zoned address allowed")
	}
	if err := strict.CheckIP("", netip.MustParseAddr("1.1.1.1")); err != nil {
		t.Error(err)
	}
}

func TestPolicyCheckHost(t *testing.T) {
	strict := Policy{}
	for _, h := range []string{"localhost", "LOCALHOST.", "api.localhost", "127.0.0.1", "::1"} {
		if err := strict.CheckHost(h); !errors.Is(err, ErrBlocked) {
			t.Errorf("CheckHost(%q) = %v", h, err)
		}
	}
	if err := strict.CheckHost("example.com"); err != nil {
		t.Error(err)
	}
	if err := (Policy{AllowPrivate: true}).CheckHost("localhost"); err != nil {
		t.Error(err)
	}
}

func TestBlockedErrorMessage(t *testing.T) {
	e := &BlockedError{Host: "evil.example", IP: netip.MustParseAddr("10.0.0.5"), Class: ClassPrivate}
	if msg := e.Error(); !strings.Contains(msg, "evil.example resolves to 10.0.0.5") || !strings.Contains(msg, "--allow-private") {
		t.Errorf("message: %s", msg)
	}
	e = &BlockedError{IP: netip.MustParseAddr("169.254.169.254"), Class: ClassMetadata}
	if msg := e.Error(); strings.Contains(msg, "--allow-private") {
		t.Errorf("metadata message must not suggest --allow-private: %s", msg)
	}
	e = &BlockedError{Host: "localhost", Class: ClassLoopback}
	if msg := e.Error(); !strings.Contains(msg, `host "localhost"`) {
		t.Errorf("message: %s", msg)
	}
	for c := ClassPublic; c <= ClassReserved+1; c++ {
		if c.String() == "" {
			t.Errorf("class %d has no name", c)
		}
	}
	if err := (Policy{}).checkSocketAddr("garbage"); !errors.Is(err, ErrBlocked) {
		t.Error("unparsable socket address allowed")
	}
}
