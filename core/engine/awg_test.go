package engine

import (
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

const awgConf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.8.0.2/32
DNS = 9.9.9.9
MTU = 1400
Jc = 4
H1 = 5

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 127.0.0.1:51820
`

func TestAWGTunRequest(t *testing.T) {
	res, err := importer.Import(awgConf)
	if err != nil {
		t.Fatal(err)
	}
	req := awgTunRequest(res.Profiles[0], singbox.Options{ExcludePackages: []string{"com.android.chrome"}})
	if req.MTU != 1400 || !slices.Equal(req.Inet4Address, []string{"10.8.0.2/32"}) ||
		!slices.Equal(req.DNSServers, []string{"9.9.9.9"}) {
		t.Fatalf("request: %+v", req)
	}
	// Local networks bypass the VPN; the tunnel's own address stays routed.
	routed := func(ip string) bool {
		a := netip.MustParseAddr(ip)
		for _, r := range singbox.ParsePrefixes(req.Inet4RouteRange) {
			if r.Contains(a) {
				return true
			}
		}
		return false
	}
	for ip, want := range map[string]bool{"8.8.8.8": true, "9.9.9.9": true, "10.8.0.2": true,
		"192.168.1.1": false, "10.0.0.5": false, "172.20.1.1": false, "169.254.1.1": false} {
		if routed(ip) != want {
			t.Errorf("%s routed=%v, want %v (routes %v)", ip, routed(ip), want, req.Inet4RouteRange)
		}
	}
	// With "proxy private" on, everything goes through the tunnel.
	all := awgTunRequest(res.Profiles[0], singbox.Options{ProxyPrivate: true})
	if !slices.Equal(all.Inet4RouteRange, []string{"0.0.0.0/0"}) {
		t.Fatalf("proxy private routes: %v", all.Inet4RouteRange)
	}
	// No IPv6 address, so IPv6 must not be routed into the tunnel.
	if len(req.Inet6RouteRange) != 0 {
		t.Fatalf("ipv6 routes without an ipv6 address: %v", req.Inet6RouteRange)
	}
	if !slices.Equal(req.ExcludePackage, []string{"com.android.chrome"}) {
		t.Fatalf("exclude: %v", req.ExcludePackage)
	}
}

type fakeTun struct{ err error }

func (f fakeTun) OpenTun(TunRequest) (int, error) { return -1, f.err }
func (f fakeTun) Protect(int) error               { return nil }

func TestAWGNeedsVPNMode(t *testing.T) {
	res, err := importer.Import(awgConf)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	e := New(Config{})
	err = e.Start(res.Profiles, singbox.Options{Selected: p.ID})
	if err == nil || !strings.Contains(err.Error(), "VPN mode") {
		t.Fatalf("err = %v", err)
	}

	var states []State
	e = New(Config{Tun: fakeTun{errors.New("no permission")}, OnState: func(s State) { states = append(states, s) }})
	err = e.Start(res.Profiles, singbox.Options{Mode: singbox.ModeTUN, Selected: p.ID})
	if err == nil || !strings.Contains(err.Error(), "no permission") {
		t.Fatalf("err = %v", err)
	}
	last := states[len(states)-1]
	if last.Status != StatusError || last.Progress == nil || last.Progress.Stage != StageTunnelUp {
		t.Fatalf("state: %+v", last)
	}
}

// A failover that had to restart the engine is reported from the profile the
// user picked, even when later hops failed too.
func TestCarriedFailoverKeepsOrigin(t *testing.T) {
	var last State
	e := New(Config{OnState: func(s State) { last = s }})
	e.mu.Lock()
	e.carryFailover = &FailoverEvent{From: "awg", To: "dead", Reason: "no handshake"}
	e.setState(State{Status: StatusStarting, Selected: "vless", Failover: &FailoverEvent{From: "dead", To: "vless", Reason: "refused"}})
	e.mu.Unlock()
	if f := last.Failover; f == nil || f.From != "awg" || f.To != "vless" || f.Reason != "no handshake" {
		t.Fatalf("failover = %+v", last.Failover)
	}
	if e.carryFailover != nil {
		t.Fatal("carried failover not consumed")
	}
}
