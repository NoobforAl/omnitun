package hostproto

import (
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/classicvpn"
	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/store"
)

// chanDevice is a fake TUN: the test injects packets and collects what the
// tunnel delivers.
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
	p := make([]byte, 36)
	p[0], p[8], p[9] = 0x45, 64, 1
	binary.BigEndian.PutUint16(p[2:], uint16(len(p)))
	copy(p[12:16], src.To4())
	copy(p[16:20], dst.To4())
	binary.BigEndian.PutUint16(p[10:], checksum(p[:20]))
	p[20] = 8
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

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-host-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 40 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

var alice = protocols.Account{Name: "alice", Password: "alice-password"}

func env() protocols.Env {
	return protocols.Env{Name: "Lab", Host: "127.0.0.1", SelfSigned: true, IKEv2PSK: "ike-psk"}
}

// serverDevices makes the in-process servers use fake TUN devices.
func serverDevices(t *testing.T) {
	t.Helper()
	classicvpn.ServerOpener = func(classicvpn.Interface) (classicvpn.Device, error) {
		return pingResponder{newChanDevice()}, nil
	}
	t.Cleanup(func() { classicvpn.ServerOpener = nil })
}

// pingThrough connects with the registry-generated profile and checks an
// ICMP echo crosses the tunnel and back.
func pingThrough(t *testing.T, key string, port int, password string) error {
	t.Helper()
	proto, _ := protocols.Get(key)
	acct := alice
	acct.Password = password
	p, err := proto.Profile(env(), port, acct)
	if err != nil {
		t.Fatal(err)
	}
	dev := newChanDevice()
	var got classicvpn.Interface
	tun, err := classicvpn.Dial(t.Context(), p, func(req classicvpn.Interface) (classicvpn.Device, error) { got = req; return dev, nil }, t.Logf)
	if err != nil {
		return err
	}
	defer tun.Close()
	if got.IPv4 == nil {
		t.Fatalf("no address negotiated: %+v", got)
	}
	gateway := net.ParseIP("10.9.0.1")
	if key == "ikev2" {
		gateway = net.ParseIP("10.10.10.1")
	}
	dev.in <- echoRequest(got.IPv4, gateway, 7)
	select {
	case r := <-dev.out:
		if len(r) < 28 || r[20] != 0 || !net.IP(r[12:16]).Equal(gateway) {
			t.Fatalf("unexpected reply % x", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no ping reply through the tunnel")
	}
	return nil
}

func waitTCP(t *testing.T, port int) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond); err == nil {
			c.Close()
			return
		}
	}
	t.Fatalf("nothing listens on port %d", port)
}

func TestSSTPServerAndGeneratedProfile(t *testing.T) {
	serverDevices(t)
	dir := tempDir(t)
	cert, key := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	if err := store.WriteSelfSignedCert(cert, key, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	certPEM, _ := os.ReadFile(cert)
	keyPEM, _ := os.ReadFile(key)
	port := freeBoth(t)
	srv, err := Start("sstp", Config{Env: env(), Port: port, Users: []protocols.Account{alice}, DataDir: dir, CertPEM: certPEM, KeyPEM: keyPEM})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	if srv.Pool() != protocols.SSTPPool {
		t.Fatalf("pool = %s", srv.Pool())
	}
	waitTCP(t, port)

	if err := pingThrough(t, "sstp", port, alice.Password); err != nil {
		t.Fatalf("the generated SSTP profile did not connect: %v", err)
	}
	if err := pingThrough(t, "sstp", port, "wrong-password"); err == nil {
		t.Fatal("SSTP accepted a wrong password")
	}
}

func TestIKEv2ServerAndGeneratedProfile(t *testing.T) {
	serverDevices(t)
	dir := tempDir(t)
	port := freeBoth(t)
	udp, err := net.ListenPacket("udp", "127.0.0.1:0") // a free UDP port standing in for 500
	if err != nil {
		t.Fatal(err)
	}
	ike := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()
	srv, err := Start("ikev2", Config{Env: env(), Port: port, IKEPort: ike, Users: []protocols.Account{alice}, DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	waitTCP(t, port) // the TCP side (IKE and ESP over TCP) is what the profile uses

	// The user list is private while the server runs and gone afterwards.
	users := filepath.Join(dir, "ikev2-users")
	if data, err := os.ReadFile(users); err != nil || string(data) != "alice:alice-password\n" {
		t.Fatalf("user list: %q %v", data, err)
	}
	if err := pingThrough(t, "ikev2", port, alice.Password); err != nil {
		t.Fatalf("the generated IKEv2 profile did not connect: %v", err)
	}
	if err := pingThrough(t, "ikev2", port, "wrong-password"); err == nil {
		t.Fatal("IKEv2 accepted a wrong password")
	}
	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(users); !os.IsNotExist(err) {
		t.Fatal("the user list (secrets) was left behind")
	}
}

func TestStartRefusesWhatItCannotServe(t *testing.T) {
	if _, err := Start("sstp", Config{Env: env(), Port: 1}); err == nil {
		t.Error("started without users")
	}
	if _, err := Start("sstp", Config{Env: env(), Port: 1, Users: []protocols.Account{alice}}); err == nil {
		t.Error("SSTP started without a certificate")
	}
	e := env()
	e.IKEv2PSK = ""
	if _, err := Start("ikev2", Config{Env: e, Port: 1, Users: []protocols.Account{alice}, DataDir: t.TempDir()}); err == nil {
		t.Error("IKEv2 started without a pre-shared key")
	}
	bad := protocols.Account{Name: "bob", Password: "a:b"}
	if _, err := Start("ikev2", Config{Env: env(), Port: 1, Users: []protocols.Account{bad}, DataDir: t.TempDir()}); err == nil {
		t.Error("IKEv2 wrote a password with a colon into its user list")
	}
	if _, err := Start("trojan", Config{Env: env(), Users: []protocols.Account{alice}}); err == nil {
		t.Error("started a protocol that sing-box serves")
	}
}
