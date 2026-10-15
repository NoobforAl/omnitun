package singbox

import (
	"net/netip"
	"strings"

	"go4.org/netipx"

	"github.com/NoobforAl/omnitun/core/profile"
)

// LocalPrefixes are the local-network ranges that bypass the VPN by default
// (Options.ProxyPrivate = false): private LANs, link-local, multicast and
// broadcast. They are kept out of the VPN's routes, so LAN devices (printers,
// routers, casting, local servers) keep working while connected.
var LocalPrefixes = []netip.Prefix{
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("255.255.255.255/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

// tunAddresses is the sing-box TUN's own subnet; its DNS server (.2) lives
// there, so it must stay inside the VPN.
var tunAddresses = []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"}

// BypassPrefixes returns LocalPrefixes minus keep: ranges a tunnel really
// routes (its own subnet, DNS servers, WireGuard AllowedIPs) stay in the VPN.
func BypassPrefixes(keep []netip.Prefix) []netip.Prefix {
	var b netipx.IPSetBuilder
	for _, p := range LocalPrefixes {
		b.AddPrefix(p)
	}
	for _, p := range keep {
		if p.IsValid() {
			b.RemovePrefix(p.Masked())
		}
	}
	set, err := b.IPSet()
	if err != nil {
		return LocalPrefixes
	}
	return set.Prefixes()
}

// SubtractPrefixes returns routes minus exclude (both may mix IPv4 and IPv6).
func SubtractPrefixes(routes, exclude []netip.Prefix) []netip.Prefix {
	var b netipx.IPSetBuilder
	for _, p := range routes {
		b.AddPrefix(p.Masked())
	}
	for _, p := range exclude {
		b.RemovePrefix(p.Masked())
	}
	set, err := b.IPSet()
	if err != nil {
		return routes
	}
	return set.Prefixes()
}

// ParsePrefixes parses CIDRs or bare addresses (as host prefixes), skipping bad ones.
func ParsePrefixes(values []string) []netip.Prefix {
	var out []netip.Prefix
	for _, v := range values {
		v = strings.TrimSpace(v)
		if p, err := netip.ParsePrefix(v); err == nil {
			out = append(out, p.Masked())
		} else if a, err := netip.ParseAddr(v); err == nil {
			out = append(out, netip.PrefixFrom(a, a.BitLen()))
		}
	}
	return out
}

// isDefaultRoute reports whether p is 0.0.0.0/0 or ::/0.
func isDefaultRoute(p netip.Prefix) bool { return p.Bits() == 0 }

// profileKeep lists private ranges the profiles route on purpose (WireGuard
// AllowedIPs narrower than a default route), plus the TUN's own subnet.
func profileKeep(profiles []*profile.Profile) []netip.Prefix {
	keep := ParsePrefixes(tunAddresses)
	for _, p := range profiles {
		if p.WireGuard == nil {
			continue
		}
		for _, a := range ParsePrefixes(p.WireGuard.AllowedIPs) {
			if !isDefaultRoute(a) {
				keep = append(keep, a)
			}
		}
		keep = append(keep, ParsePrefixes(p.WireGuard.DNS)...)
	}
	return keep
}

func prefixStrings(ps []netip.Prefix) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}
