// Package netguard enforces CrawlGrade's network policy: which IP addresses
// the crawler may connect to. It protects users against server-side request
// forgery, where a site redirects or resolves the crawler to loopback,
// private networks or cloud metadata services.
//
// The policy is enforced in the dialer (see ADR 0001), so it applies to every
// request and every redirect hop, and the address that was checked is the
// address that is connected to.
package netguard

import (
	"net/netip"
)

// Class describes the kind of network an address belongs to.
type Class int

// Address classes.
const (
	ClassPublic Class = iota
	ClassLoopback
	ClassPrivate
	ClassLinkLocal
	ClassSharedAddress // RFC 6598 carrier-grade NAT
	ClassMetadata
	ClassUnspecified
	ClassMulticast
	ClassBroadcast
	ClassReserved
)

func (c Class) String() string {
	switch c {
	case ClassPublic:
		return "public"
	case ClassLoopback:
		return "loopback"
	case ClassPrivate:
		return "private network"
	case ClassLinkLocal:
		return "link-local"
	case ClassSharedAddress:
		return "carrier-grade NAT"
	case ClassMetadata:
		return "cloud metadata service"
	case ClassUnspecified:
		return "unspecified"
	case ClassMulticast:
		return "multicast"
	case ClassBroadcast:
		return "broadcast"
	case ClassReserved:
		return "reserved"
	}
	return "unknown"
}

// AllowedWithPrivate reports whether --allow-private permits the class.
func (c Class) AllowedWithPrivate() bool {
	switch c {
	case ClassPublic, ClassLoopback, ClassPrivate, ClassLinkLocal, ClassSharedAddress:
		return true
	}
	return false
}

var metadataAddrs = []netip.Addr{
	netip.MustParseAddr("169.254.169.254"), // AWS, GCP, Azure, OpenStack, DigitalOcean, ...
	netip.MustParseAddr("169.254.170.2"),   // AWS ECS task metadata
	netip.MustParseAddr("100.100.100.200"), // Alibaba Cloud
	netip.MustParseAddr("fd00:ec2::254"),   // AWS IMDS over IPv6
}

type prefixClass struct {
	prefix netip.Prefix
	class  Class
}

func pc(s string, c Class) prefixClass { return prefixClass{netip.MustParsePrefix(s), c} }

var v4Prefixes = []prefixClass{
	pc("0.0.0.0/8", ClassUnspecified),
	pc("10.0.0.0/8", ClassPrivate),
	pc("100.64.0.0/10", ClassSharedAddress),
	pc("127.0.0.0/8", ClassLoopback),
	pc("169.254.0.0/16", ClassLinkLocal),
	pc("172.16.0.0/12", ClassPrivate),
	pc("192.0.0.0/24", ClassReserved),   // IETF protocol assignments
	pc("192.0.2.0/24", ClassReserved),   // TEST-NET-1
	pc("192.88.99.0/24", ClassReserved), // deprecated 6to4 relay anycast
	pc("192.168.0.0/16", ClassPrivate),
	pc("198.18.0.0/15", ClassReserved),   // benchmarking
	pc("198.51.100.0/24", ClassReserved), // TEST-NET-2
	pc("203.0.113.0/24", ClassReserved),  // TEST-NET-3
	pc("224.0.0.0/4", ClassMulticast),
	pc("255.255.255.255/32", ClassBroadcast),
	pc("240.0.0.0/4", ClassReserved),
}

var v6Prefixes = []prefixClass{
	pc("::/128", ClassUnspecified),
	pc("::1/128", ClassLoopback),
	pc("::/96", ClassReserved),         // deprecated IPv4-compatible
	pc("64:ff9b:1::/48", ClassPrivate), // local-use NAT64
	pc("100::/64", ClassReserved),      // discard-only
	pc("2001::/32", ClassReserved),     // Teredo: embedded client address is obfuscated
	pc("2001:10::/28", ClassReserved),  // deprecated ORCHID
	pc("2001:20::/28", ClassReserved),  // ORCHIDv2
	pc("2001:db8::/32", ClassReserved), // documentation
	pc("3fff::/20", ClassReserved),     // documentation (RFC 9637)
	pc("5f00::/16", ClassReserved),     // SRv6 SIDs
	pc("fc00::/7", ClassPrivate),       // unique local
	pc("fe80::/10", ClassLinkLocal),
	pc("fec0::/10", ClassPrivate), // deprecated site-local
	pc("ff00::/8", ClassMulticast),
}

var (
	nat64Prefix   = netip.MustParsePrefix("64:ff9b::/96")
	sixToFour     = netip.MustParsePrefix("2002::/16")
	globalUnicast = netip.MustParsePrefix("2000::/3")
)

// Classify returns the class of ip. Addresses that embed an IPv4 address
// (IPv4-mapped, NAT64, 6to4) are classified by the embedded address.
func Classify(ip netip.Addr) Class {
	if !ip.IsValid() {
		return ClassReserved
	}
	ip = ip.WithZone("")
	if ip.Is4In6() {
		ip = ip.Unmap()
	}
	for _, m := range metadataAddrs {
		if ip == m {
			return ClassMetadata
		}
	}
	if ip.Is4() {
		for _, e := range v4Prefixes {
			if e.prefix.Contains(ip) {
				return e.class
			}
		}
		return ClassPublic
	}
	b := ip.As16()
	switch {
	case nat64Prefix.Contains(ip):
		return Classify(netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	case sixToFour.Contains(ip):
		return Classify(netip.AddrFrom4([4]byte{b[2], b[3], b[4], b[5]}))
	}
	for _, e := range v6Prefixes {
		if e.prefix.Contains(ip) {
			return e.class
		}
	}
	if !globalUnicast.Contains(ip) {
		return ClassReserved
	}
	return ClassPublic
}
