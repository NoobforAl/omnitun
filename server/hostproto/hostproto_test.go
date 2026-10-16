package hostproto

import (
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// testCertPEM is the certificate of the server under test; IKEv2 profiles
// pin it.
var testCertPEM string

func env() protocols.Env {
	return protocols.Env{Name: "Lab", Host: "127.0.0.1", SelfSigned: true, CertPEM: testCertPEM}
}

// certFiles makes a self-signed certificate in dir and remembers it for
// generated profiles.
func certFiles(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	certFile, keyFile = filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	if err := store.WriteSelfSignedCert(certFile, keyFile, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	testCertPEM = string(data)
	t.Cleanup(func() { testCertPEM = "" })
	return certFile, keyFile
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
	return pingThroughWith(t, key, port, password, nil)
}

// pingThroughWith is pingThrough that calls during, while the tunnel is
// still up, after the reply arrived.
func pingThroughWith(t *testing.T, key string, port int, password string, during func()) error {
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
	if during != nil {
		during()
	}
	return nil
}

// usageOf is what a host server counted for one user.
func usageOf(t *testing.T, srv Server, user string) Usage {
	t.Helper()
	m, ok := srv.(Meter)
	if !ok {
		t.Fatalf("%T does not count traffic per user", srv)
	}
	for _, u := range m.Usage() {
		if u.User == user {
			return u
		}
	}
	return Usage{User: user}
}

// A user's traffic is counted: both directions while connected, and what was
// counted stays after they leave.
func TestSSTPCountsTrafficPerUser(t *testing.T) {
	serverDevices(t)
	dir := tempDir(t)
	cert, key := certFiles(t, dir)
	certPEM, _ := os.ReadFile(cert)
	keyPEM, _ := os.ReadFile(key)
	port := freeBoth(t)
	srv, err := Start("sstp", Config{Env: env(), Port: port, Users: []protocols.Account{alice}, DataDir: dir, CertPEM: certPEM, KeyPEM: keyPEM})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	waitTCP(t, port)
	if u := usageOf(t, srv, "alice"); u.Up != 0 || u.Down != 0 {
		t.Fatalf("traffic before anyone connected: %+v", u)
	}
	if err := pingThrough(t, "sstp", port, alice.Password); err != nil {
		t.Fatal(err)
	}
	u := usageOf(t, srv, "alice")
	if u.Up < 28 || u.Down < 28 { // an echo request and its reply, 28 bytes each at least
		t.Fatalf("usage after a ping: %+v", u)
	}
	again := usageOf(t, srv, "alice")
	if again.Up < u.Up || again.Down < u.Down {
		t.Fatalf("counted traffic went down: %+v then %+v", u, again)
	}
	if other := usageOf(t, srv, "bob"); other.Up != 0 || other.Down != 0 {
		t.Fatalf("someone else was charged: %+v", other)
	}
}

func TestIKEv2CountsTrafficPerUser(t *testing.T) {
	serverDevices(t)
	dir := tempDir(t)
	certFile, keyFile := certFiles(t, dir)
	port := freeBoth(t)
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ike := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()
	srv, err := Start("ikev2", Config{Env: env(), Port: port, IKEPort: ike, CertFile: certFile, KeyFile: keyFile, Users: []protocols.Account{alice}, DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	waitTCP(t, port)

	var live Usage
	err = pingThroughWith(t, "ikev2", port, alice.Password, func() {
		// The peer's counters are read while its tunnel exists.
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			if live = usageOf(t, srv, "alice"); live.Up > 0 && live.Down > 0 {
				return
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if live.Up < 28 || live.Down < 28 {
		t.Fatalf("usage while connected: %+v", live)
	}
	// After the user left, what was counted is still there.
	time.Sleep(200 * time.Millisecond)
	if after := usageOf(t, srv, "alice"); after.Up < live.Up || after.Down < live.Down {
		t.Fatalf("usage forgotten after disconnect: %+v then %+v", live, after)
	}
}

func TestIKEMeterSumsChangesNotSnapshots(t *testing.T) {
	m := &ikeMeter{last: map[string][2]uint64{}, total: map[string][2]uint64{}}
	key := "alice\x00" + "10.10.10.2"
	get := func(cur map[string][2]uint64) Usage {
		for _, u := range m.add(cur) {
			if u.User == "alice" {
				return u
			}
		}
		t.Fatal("alice missing")
		return Usage{}
	}
	if u := get(map[string][2]uint64{key: {100, 40}}); u.Up != 100 || u.Down != 40 {
		t.Fatalf("first reading: %+v", u)
	}
	if u := get(map[string][2]uint64{key: {150, 90}}); u.Up != 150 || u.Down != 90 {
		t.Fatalf("second reading: %+v", u)
	}
	// The tunnel ended; the same address is taken again by a new one that starts at zero.
	if u := get(map[string][2]uint64{}); u.Up != 150 || u.Down != 90 {
		t.Fatalf("no peer: %+v", u)
	}
	if u := get(map[string][2]uint64{key: {10, 5}}); u.Up != 160 || u.Down != 95 {
		t.Fatalf("new tunnel: %+v", u)
	}
	// A shrinking counter at the same address is a new tunnel too.
	get(map[string][2]uint64{key: {50, 20}})
	if u := get(map[string][2]uint64{key: {3, 1}}); u.Up != 203 || u.Down != 111 {
		t.Fatalf("restarted counter: %+v", u)
	}
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
	cert, key := certFiles(t, dir)
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
	certFile, keyFile := certFiles(t, dir)
	port := freeBoth(t)
	udp, err := net.ListenPacket("udp", "127.0.0.1:0") // a free UDP port standing in for 500
	if err != nil {
		t.Fatal(err)
	}
	ike := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()
	srv, err := Start("ikev2", Config{Env: env(), Port: port, IKEPort: ike, CertFile: certFile, KeyFile: keyFile, Users: []protocols.Account{alice}, DataDir: dir})
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
	if _, err := Start("ikev2", Config{Env: env(), Port: 1, Users: []protocols.Account{alice}, DataDir: t.TempDir()}); err == nil {
		t.Error("IKEv2 started without a certificate")
	}
	dir := t.TempDir()
	certFile, keyFile := certFiles(t, dir)
	bad := protocols.Account{Name: "bob", Password: "a:b"}
	if _, err := Start("ikev2", Config{Env: env(), Port: 1, CertFile: certFile, KeyFile: keyFile, Users: []protocols.Account{bad}, DataDir: t.TempDir()}); err == nil {
		t.Error("IKEv2 wrote a password with a colon into its user list")
	}
	if _, err := Start("trojan", Config{Env: env(), Users: []protocols.Account{alice}}); err == nil {
		t.Error("started a protocol that sing-box serves")
	}
}

// A client that pins one certificate refuses a server presenting another.
func TestIKEv2ClientRefusesAnotherCertificate(t *testing.T) {
	serverDevices(t)
	serverDir, otherDir := tempDir(t), tempDir(t)
	certFile, keyFile := certFiles(t, serverDir)
	port := freeBoth(t)
	udp, _ := net.ListenPacket("udp", "127.0.0.1:0")
	ike := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()
	srv, err := Start("ikev2", Config{Env: env(), Port: port, IKEPort: ike, CertFile: certFile, KeyFile: keyFile, Users: []protocols.Account{alice}, DataDir: serverDir})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	waitTCP(t, port)
	certFiles(t, otherDir) // the profile now pins a different certificate
	if err := pingThrough(t, "ikev2", port, alice.Password); err == nil {
		t.Fatal("the client connected to a server whose certificate it had not pinned")
	}
}

// The servers open their TUN device but never address it: the host has to, or
// the device stays down (found by running the server on real Linux).
func TestConfigureTUN(t *testing.T) {
	r := &fakeRunner{}
	_, network, _ := net.ParseCIDR("10.9.0.0/24")
	if err := configureTUN(r, "tun7", net.ParseIP("10.9.0.1"), network); err != nil {
		t.Fatal(err)
	}
	want := []string{"ip addr add 10.9.0.1/24 dev tun7", "ip link set dev tun7 up"}
	if strings.Join(r.ran, "|") != strings.Join(want, "|") {
		t.Fatalf("ran %v, want %v", r.ran, want)
	}
	// A test device has no name: nothing to do, nothing run.
	r = &fakeRunner{}
	if err := configureTUN(r, "", net.ParseIP("10.9.0.1"), network); err != nil || len(r.ran) != 0 {
		t.Fatalf("unnamed device: %v %v", err, r.ran)
	}
	// A failing command is an error: a server whose device is down must not look started.
	r = &fakeRunner{failing: map[string]bool{"ip link set dev tun7 up": true}}
	if err := configureTUN(r, "tun7", net.ParseIP("10.9.0.1"), network); err == nil {
		t.Fatal("a failed `ip link set up` was ignored")
	}
}
