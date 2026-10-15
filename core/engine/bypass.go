package engine

import (
	"net/netip"

	"github.com/NoobforAl/omnitun/core/singbox"
)

// applyLocalBypass removes local-network ranges from req's routes so LAN
// traffic never enters the VPN. keep are ranges the tunnel itself needs (its
// address, DNS servers, explicitly routed private subnets).
func applyLocalBypass(req *TunRequest, keep []netip.Prefix) {
	keep = append(keep, singbox.ParsePrefixes(req.Inet4Address)...)
	keep = append(keep, singbox.ParsePrefixes(req.Inet6Address)...)
	keep = append(keep, singbox.ParsePrefixes(req.DNSServers)...)
	exclude := singbox.BypassPrefixes(keep)
	split := func(ranges []string, v4 bool) []string {
		out := []string{}
		for _, p := range singbox.SubtractPrefixes(singbox.ParsePrefixes(ranges), exclude) {
			if p.Addr().Is4() == v4 {
				out = append(out, p.String())
			}
		}
		return out
	}
	req.Inet4RouteRange = split(req.Inet4RouteRange, true)
	req.Inet6RouteRange = split(req.Inet6RouteRange, false)
}
