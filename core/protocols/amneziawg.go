package protocols

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// AmneziaWG network: the server is .1 and users get their WireGuard address
// moved from 10.30.0.0/16 into it.
const (
	AWGPool          = "10.31.0.0/16"
	awgServerAddress = "10.31.0.1/16"
)

// AmneziaWG is the server's AmneziaWG identity and obfuscation parameters.
type AmneziaWG struct {
	PrivateKey, PublicKey string
	// Params are jc, jmin, jmax, s1, s2, h1-h4 (lower-case keys, decimal values).
	Params map[string]string
}

// AWGServerAddress is the server's address inside the tunnel.
func AWGServerAddress() string { return awgServerAddress }

// AWGAddress maps a user's WireGuard address (10.30.a.b/32) to their
// AmneziaWG address (10.31.a.b/32).
func AWGAddress(wgAddress string) (string, error) {
	p, err := netip.ParsePrefix(wgAddress)
	if err != nil || !p.Addr().Is4() {
		return "", fmt.Errorf("%q is not an IPv4 tunnel address", wgAddress)
	}
	b := p.Addr().As4()
	b[1] = 31
	return netip.PrefixFrom(netip.AddrFrom4(b), 32).String(), nil
}

type amneziawg struct{ hostBase }

func (amneziawg) Spec() Spec {
	return Spec{
		Key: "amneziawg", Name: "AmneziaWG", Label: "AmneziaWG", Profile: profile.AmneziaWG,
		Description: "WireGuard with junk packets and changed headers, to get past DPI that blocks plain WireGuard. Uses each user's WireGuard key. Needs Linux and root/CAP_NET_ADMIN. UDP only.",
		Transport:   "udp", DefaultPort: 51821, ShareLink: true, FileExt: ".conf",
		Host: true, DefaultOff: true,
	}
}

func (a amneziawg) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	if u.WGPrivateKey == "" || u.WGAddress == "" {
		return nil, fmt.Errorf("user %s has no WireGuard key or address", u.Name)
	}
	if env.AWG.PublicKey == "" || len(env.AWG.Params) == 0 {
		return nil, fmt.Errorf("the server has no AmneziaWG identity")
	}
	addr, err := AWGAddress(u.WGAddress)
	if err != nil {
		return nil, err
	}
	params := make(map[string]string, len(env.AWG.Params))
	for k, v := range env.AWG.Params {
		params[k] = v
	}
	return &profile.Profile{
		Name: profileName(env, a.Spec()), Protocol: profile.AmneziaWG, Server: env.Host, Port: port,
		WireGuard: &profile.WG{
			PrivateKey: u.WGPrivateKey, PeerPublicKey: env.AWG.PublicKey,
			Address: []string{addr}, AllowedIPs: []string{"0.0.0.0/0", "::/0"},
			DNS: []string{"1.1.1.1", "8.8.8.8"}, MTU: wireguardMTU, Keepalive: 25, Amnezia: params,
		},
	}, nil
}

// File is the user's AmneziaWG .conf (WireGuard's format plus the obfuscation
// parameters), importable by the AmneziaWG and Amnezia VPN apps.
func (a amneziawg) File(env Env, port int, u Account) (string, string, error) {
	p, err := a.Profile(env, port, u)
	if err != nil {
		return "", "", err
	}
	g := p.WireGuard
	var b strings.Builder
	fmt.Fprintf(&b, "# %s (AmneziaWG) for %s\n[Interface]\nPrivateKey = %s\nAddress = %s\nDNS = %s\nMTU = %d\n",
		env.Name, u.Name, g.PrivateKey, strings.Join(g.Address, ", "), strings.Join(g.DNS, ", "), g.MTU)
	keys := make([]string, 0, len(g.Amnezia))
	for k := range g.Amnezia {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// Config files spell them Jc, Jmin, ... S1, H1.
		fmt.Fprintf(&b, "%s = %s\n", strings.ToUpper(k[:1])+k[1:], g.Amnezia[k])
	}
	fmt.Fprintf(&b, "\n[Peer]\nPublicKey = %s\nAllowedIPs = %s\nEndpoint = %s:%d\nPersistentKeepalive = %d\n",
		g.PeerPublicKey, strings.Join(g.AllowedIPs, ", "), endpointHost(env.Host), port, g.Keepalive)
	return env.Name + "-amneziawg.conf", b.String(), nil
}
