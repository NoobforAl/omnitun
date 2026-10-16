package classicvpn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xen0bit/veepin/ikev2"
	"github.com/xen0bit/veepin/sstp"

	"github.com/NoobforAl/omnitun/core/profile"
)

// chanDevice is a fake TUN: the test injects packets the "apps" send and
// collects packets the tunnel delivers.
type chanDevice struct {
	in, out chan []byte
	closed  chan struct{}
}

func newChanDevice() *chanDevice {
	return &chanDevice{in: make(chan []byte, 16), out: make(chan []byte, 16), closed: make(chan struct{})}
}

func (d *chanDevice) Read(b []byte) (int, error) {
	select {
	case p := <-d.in:
		return copy(b, p), nil
	case <-d.closed:
		return 0, os.ErrClosed
	}
}

func (d *chanDevice) Write(p []byte) (int, error) {
	select {
	case d.out <- append([]byte(nil), p...):
	default:
	}
	return len(p), nil
}

func (d *chanDevice) Close() error {
	select {
	case <-d.closed:
	default:
		close(d.closed)
	}
	return nil
}

// pingResponder is the server's fake TUN: it answers ICMP echo requests.
type pingResponder struct{ *chanDevice }

func (r pingResponder) Write(p []byte) (int, error) {
	if reply := echoReply(p); reply != nil {
		r.in <- reply
	}
	return len(p), nil
}

func checksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(b); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(b[i:]))
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}
	return ^uint16(sum)
}

func echoRequest(src, dst net.IP, id uint16) []byte {
	p := make([]byte, 28+8)
	p[0], p[8], p[9] = 0x45, 64, 1
	binary.BigEndian.PutUint16(p[2:], uint16(len(p)))
	copy(p[12:16], src.To4())
	copy(p[16:20], dst.To4())
	binary.BigEndian.PutUint16(p[10:], checksum(p[:20]))
	p[20] = 8 // echo request
	binary.BigEndian.PutUint16(p[24:], id)
	copy(p[28:], "omnitun!")
	binary.BigEndian.PutUint16(p[22:], checksum(p[20:]))
	return p
}

func echoReply(p []byte) []byte {
	if len(p) < 28 || p[0]>>4 != 4 || p[9] != 1 || p[20] != 8 {
		return nil
	}
	r := append([]byte(nil), p...)
	copy(r[12:16], p[16:20])
	copy(r[16:20], p[12:16])
	r[10], r[11] = 0, 0
	binary.BigEndian.PutUint16(r[10:], checksum(r[:20]))
	r[20], r[22], r[23] = 0, 0, 0
	binary.BigEndian.PutUint16(r[22:], checksum(r[20:]))
	return r
}

func selfSigned(t *testing.T) (certPEM, keyPEM []byte) {
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "server.test"},
		DNSNames: []string{"server.test"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(k)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func freeTCP(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// waitTCP waits until something accepts connections on the local port, so a
// test doesn't dial before its server goroutine has started listening.
func waitTCP(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond); err == nil {
			c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not start listening on port %d", port)
}

// freeBoth returns a port that is free for both TCP and UDP. On Windows a
// port that is free for one can sit inside a reserved range for the other.
func freeBoth(t *testing.T) int {
	t.Helper()
	for range 50 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		c, err := net.ListenPacket("udp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		l.Close()
		if err == nil {
			c.Close()
			return port
		}
	}
	t.Fatal("no port free for both TCP and UDP")
	return 0
}

func freeUDP(t *testing.T) int {
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

// pingThrough dials p and checks an ICMP echo crosses the tunnel and back.
func pingThrough(t *testing.T, p *profile.Profile, server net.IP) {
	t.Helper()
	dev := newChanDevice()
	var got Interface
	tun, err := Dial(t.Context(), p, func(req Interface) (Device, error) { got = req; return dev, nil }, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()
	if got.IPv4 == nil || IPv4Prefix(got) == "" {
		t.Fatalf("interface: %+v", got)
	}
	if server == nil {
		server = net.IPv4(10, 0, 0, 1)
	}
	dev.in <- echoRequest(got.IPv4, server, 7)
	select {
	case r := <-dev.out:
		if len(r) < 28 || r[20] != 0 || !net.IP(r[12:16]).Equal(server.To4()) {
			t.Fatalf("unexpected reply % x", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no ping reply through the tunnel")
	}
	// The counter is bumped just after the device write returns, which can be
	// after the reply is already visible to the test.
	var rx, tx int64
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if rx, tx = tun.Traffic(); rx > 0 && tx > 0 {
			break
		}
	}
	if rx == 0 || tx == 0 {
		t.Fatalf("traffic rx=%d tx=%d", rx, tx)
	}
}

func useServerDevice(t *testing.T) {
	mu.Lock()
	ServerOpener = func(Interface) (Device, error) { return pingResponder{newChanDevice()}, nil }
	mu.Unlock()
	t.Cleanup(func() { mu.Lock(); ServerOpener = nil; mu.Unlock() })
}

func TestSSTP(t *testing.T) {
	useServerDevice(t)
	cert, key := selfSigned(t)
	port := freeTCP(t)
	srv, err := sstp.NewServer(sstp.ServerConfig{
		Cert: cert, Key: key, ListenIP: "127.0.0.1", ListenPort: port,
		Pool: "10.20.0.0/24", Users: map[string]string{"alice": "secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	waitTCP(t, port)

	p := &profile.Profile{Protocol: profile.SSTP, Server: "127.0.0.1", Port: port, Username: "alice", Password: "secret",
		TLS: &profile.TLS{Enabled: true, Insecure: true}}
	pingThrough(t, p, srv.Gateway())

	// A wrong password is reported as an authentication failure.
	bad := *p
	bad.Password = "wrong"
	if _, err := Dial(t.Context(), &bad, func(Interface) (Device, error) { return newChanDevice(), nil }, nil); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestIKEv2PSKOverTCP(t *testing.T) {
	useServerDevice(t)
	port500, port4500 := freeBoth(t), freeBoth(t)
	srv, err := ikev2.NewServer(ikev2.ServerConfig{
		ListenIP: "127.0.0.1", Port500: port500, Port4500: port4500,
		PSK: "psk-secret", LocalID: "server.test", Pool: "10.21.0.0/24", TCP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	waitTCP(t, port4500)

	p := &profile.Profile{Protocol: profile.IKEv2, Server: "127.0.0.1", Port: port4500,
		IKEv2: &profile.IKEv2Opts{Auth: "psk", PSK: "psk-secret", RemoteID: "server.test", TCP: true}}
	pingThrough(t, p, nil)
}

func TestIKEv2EAPOverTCP(t *testing.T) {
	useServerDevice(t)
	users := filepath.Join(t.TempDir(), "users")
	if err := os.WriteFile(users, []byte("bob:eap-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	port500, port4500 := freeBoth(t), freeBoth(t)
	srv, err := ikev2.NewServer(ikev2.ServerConfig{
		ListenIP: "127.0.0.1", Port500: port500, Port4500: port4500,
		PSK: "psk-secret", LocalID: "server.test", Pool: "10.22.0.0/24", TCP: true, EAPUsers: users,
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	waitTCP(t, port4500)

	p := &profile.Profile{Protocol: profile.IKEv2, Server: "127.0.0.1", Port: port4500, Username: "bob", Password: "eap-secret",
		IKEv2: &profile.IKEv2Opts{Auth: "eap", PSK: "psk-secret", TCP: true}}
	pingThrough(t, p, nil)

	missing := *p
	missing.Password = ""
	if _, err := Dial(t.Context(), &missing, nil, nil); !errors.Is(err, profile.ErrNeedsCredentials) {
		t.Fatalf("missing password: %v", err)
	}
}

// TestLiveSSTP connects to a real SSTP server when OMNITUN_LIVE_SSTP is set
// to user:password@host[:port] (e.g. a VPN Gate server: vpn:vpn@vpnNNN.opengw.net).
func TestLiveSSTP(t *testing.T) {
	target := os.Getenv("OMNITUN_LIVE_SSTP")
	if target == "" {
		t.Skip("set OMNITUN_LIVE_SSTP=user:password@host[:port]")
	}
	p, err := parseLiveSSTP(target)
	if err != nil {
		t.Fatal(err)
	}
	var got Interface
	dev := newChanDevice()
	tun, err := Dial(t.Context(), p, func(req Interface) (Device, error) { got = req; return dev, nil }, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()
	t.Logf("connected: address %s, DNS %v", IPv4Prefix(got), got.DNS)
	// Ping a public resolver through the tunnel (optionally after a pause).
	if d, _ := time.ParseDuration(os.Getenv("OMNITUN_LIVE_DELAY")); d > 0 {
		time.Sleep(d)
	}
	dst := net.IPv4(1, 1, 1, 1)
	for i := range 3 {
		dev.in <- echoRequest(got.IPv4, dst, uint16(i))
		deadline := time.After(3 * time.Second)
	wait:
		for {
			select {
			case r := <-dev.out:
				t.Logf("packet %d bytes %s -> %s proto %d type %d", len(r), net.IP(r[12:16]), net.IP(r[16:20]), r[9], r[20])
				if r[9] == 1 && r[20] == 0 && net.IP(r[12:16]).Equal(dst) {
					return
				}
			case <-deadline:
				break wait
			}
		}
	}
	rx, tx := tun.Traffic()
	t.Fatalf("no ping reply (rx=%d tx=%d)", rx, tx)
}

func parseLiveSSTP(s string) (*profile.Profile, error) {
	creds, hostPort, _ := strings.Cut(s, "@")
	user, pass, _ := strings.Cut(creds, ":")
	host, port := hostPort, 443
	if h, ps, err := net.SplitHostPort(hostPort); err == nil {
		host = h
		port, _ = strconv.Atoi(ps)
	}
	return &profile.Profile{Protocol: profile.SSTP, Server: host, Port: port, Username: user, Password: pass,
		TLS: &profile.TLS{Enabled: true, Insecure: true}}, nil
}
