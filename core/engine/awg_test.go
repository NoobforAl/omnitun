package engine

import (
	"errors"
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
		!slices.Equal(req.DNSServers, []string{"9.9.9.9"}) || !slices.Equal(req.Inet4RouteRange, []string{"0.0.0.0/0"}) {
		t.Fatalf("request: %+v", req)
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
