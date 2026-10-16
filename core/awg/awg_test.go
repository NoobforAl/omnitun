package awg

import (
	"errors"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"

	"github.com/NoobforAl/omnitun/core/importer"
)

const conf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.8.0.2/32
DNS = 1.1.1.1
Jc = 4
Jmin = 40
Jmax = 70
S1 = 15
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
PresharedKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.5:51820
`

func TestUAPI(t *testing.T) {
	res, err := importer.Import(conf)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	uapi, err := UAPI(p, netip.MustParseAddrPort("203.0.113.5:51820"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"private_key=c809f3e531", // hex of the decoded key
		"jc=4\njmin=40\njmax=70\ns1=15\ns2=20\nh1=1\nh2=2\nh3=3\nh4=4\n",
		"replace_peers=true\npublic_key=",
		"preshared_key=",
		"endpoint=203.0.113.5:51820\n",
		"persistent_keepalive_interval=25\n",
		"allowed_ip=0.0.0.0/0\n",
	}
	for _, w := range want {
		if !strings.Contains(uapi, w) {
			t.Errorf("UAPI missing %q:\n%s", w, uapi)
		}
	}
	// Device keys must come before the peer section.
	if strings.Index(uapi, "h4=") > strings.Index(uapi, "public_key=") {
		t.Error("device parameters after peer section")
	}
}

func TestUAPIRejectsAWG2Options(t *testing.T) {
	res, err := importer.Import(strings.Replace(conf, "H4 = 4", "H4 = 4\nS3 = 10", 1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = UAPI(res.Profiles[0], netip.MustParseAddrPort("203.0.113.5:51820"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

// AmneziaWG 1.5's signature packets and controlled junk reach the device
// configuration, and amneziawg-go accepts them.
func TestUAPICarriesAWG15Chains(t *testing.T) {
	text := strings.Replace(conf, "H4 = 4", "H4 = 4\nI1 = <b 0xc0ff00><r 16>\nJ1 = <b 0x01><r 8>", 1)
	res, err := importer.Import(text)
	if err != nil {
		t.Fatal(err)
	}
	uapi, err := UAPI(res.Profiles[0], netip.MustParseAddrPort("203.0.113.5:51820"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"i1=<b 0xc0ff00><r 16>\n", "j1=<b 0x01><r 8>\n"} {
		if !strings.Contains(uapi, want) {
			t.Errorf("UAPI missing %q:\n%s", want, uapi)
		}
	}
	if strings.Index(uapi, "i1=") > strings.Index(uapi, "public_key=") {
		t.Error("chains after the peer section")
	}
	dev := device.NewDevice(newNopTun(), conn.NewDefaultBind(), &device.Logger{Verbosef: func(string, ...any) {}, Errorf: func(string, ...any) {}})
	defer dev.Close()
	if err := dev.IpcSet(uapi); err != nil {
		t.Fatalf("amneziawg-go rejected the configuration: %v", err)
	}
}

func TestUAPIRejectsHeaderRangesAndBrokenChains(t *testing.T) {
	res, _ := importer.Import(strings.Replace(conf, "H1 = 1", "H1 = 100-200", 1))
	if _, err := UAPI(res.Profiles[0], netip.MustParseAddrPort("203.0.113.5:51820")); !errors.Is(err, ErrUnsupported) {
		t.Errorf("header range: %v, want ErrUnsupported", err)
	}
	p := *res.Profiles[0]
	wg := *p.WireGuard
	wg.Amnezia = map[string]string{"i1": "<b 0x01>\nprivate_key=00"}
	p.WireGuard = &wg
	if _, err := UAPI(&p, netip.MustParseAddrPort("203.0.113.5:51820")); err == nil {
		t.Error("a line break in a chain was passed to the device configuration")
	}
}

// nopTun is a TUN device that never has traffic.
type nopTun struct {
	events chan tun.Event
	closed chan struct{}
}

func newNopTun() *nopTun {
	t := &nopTun{events: make(chan tun.Event, 1), closed: make(chan struct{})}
	t.events <- tun.EventUp
	return t
}

func (t *nopTun) File() *os.File           { return nil }
func (t *nopTun) MTU() (int, error)        { return 1280, nil }
func (t *nopTun) Name() (string, error)    { return "nop", nil }
func (t *nopTun) Events() <-chan tun.Event { return t.events }
func (t *nopTun) BatchSize() int           { return 1 }
func (t *nopTun) Read([][]byte, []int, int) (int, error) {
	<-t.closed
	return 0, os.ErrClosed
}
func (t *nopTun) Write(bufs [][]byte, _ int) (int, error) { return len(bufs), nil }
func (t *nopTun) Close() error {
	select {
	case <-t.closed:
	default:
		close(t.closed)
	}
	return nil
}

func TestParseStats(t *testing.T) {
	s := parseStats("public_key=ab\nrx_bytes=100\ntx_bytes=50\nlast_handshake_time_sec=1700000000\nlast_handshake_time_nsec=5\n")
	if s.RxBytes != 100 || s.TxBytes != 50 || s.LastHandshake.Unix() != 1700000000 {
		t.Fatalf("stats: %+v", s)
	}
	if !parseStats("rx_bytes=0\n").LastHandshake.IsZero() {
		t.Fatal("no handshake should be zero time")
	}
}
