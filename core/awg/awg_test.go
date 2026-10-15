package awg

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

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
	res, err := importer.Import(strings.Replace(conf, "H4 = 4", "H4 = 4\nS3 = 10\nI1 = <b 0x01>", 1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = UAPI(res.Profiles[0], netip.MustParseAddrPort("203.0.113.5:51820"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
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
