package hostproto

import (
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"

	"github.com/NoobforAl/omnitun/core/awg"
	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/store"
)

// fakeTun is a tun.Device the test feeds and reads. With respond set it plays
// the host side of the tunnel and answers ICMP echo requests.
type fakeTun struct {
	in      chan []byte // packets the device reads from the "interface"
	out     chan []byte // packets the device wrote to it
	events  chan tun.Event
	closed  chan struct{}
	respond bool
}

func newFakeTun(respond bool) *fakeTun {
	t := &fakeTun{in: make(chan []byte, 16), out: make(chan []byte, 16), events: make(chan tun.Event, 1), closed: make(chan struct{}), respond: respond}
	t.events <- tun.EventUp
	return t
}

func (t *fakeTun) File() *os.File           { return nil }
func (t *fakeTun) MTU() (int, error)        { return awgMTU, nil }
func (t *fakeTun) Name() (string, error)    { return awgInterface, nil }
func (t *fakeTun) Events() <-chan tun.Event { return t.events }
func (t *fakeTun) BatchSize() int           { return 1 }

func (t *fakeTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	select {
	case p := <-t.in:
		sizes[0] = copy(bufs[0][offset:], p)
		return 1, nil
	case <-t.closed:
		return 0, os.ErrClosed
	}
}

func (t *fakeTun) Write(bufs [][]byte, offset int) (int, error) {
	for _, b := range bufs {
		p := append([]byte(nil), b[offset:]...)
		if t.respond {
			if r := echoReply(p); r != nil {
				select {
				case t.in <- r:
				default:
				}
			}
			continue
		}
		select {
		case t.out <- p:
		default:
		}
	}
	return len(bufs), nil
}

func (t *fakeTun) Close() error {
	select {
	case <-t.closed:
	default:
		close(t.closed)
		close(t.events)
	}
	return nil
}

func freeUDP(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

// fakeTunOpener makes the server use a fake TUN device for the test.
func fakeTunOpener(t *testing.T) {
	t.Helper()
	real := openTUN
	openTUN = func(string, int) (tun.Device, error) { return newFakeTun(true), nil }
	t.Cleanup(func() { openTUN = real })
}

// awgFixture is a started AmneziaWG server and the pieces to build clients.
type awgFixture struct {
	srv  Server
	port int
	env  protocols.Env
	user protocols.Account
	run  *fakeRunner
}

func awgEnv(t *testing.T) (protocols.Env, protocols.Account) {
	t.Helper()
	a, err := store.NewAmneziaWG()
	if err != nil {
		t.Fatal(err)
	}
	priv, pub, err := store.NewWireGuardKeys()
	if err != nil {
		t.Fatal(err)
	}
	e := env()
	e.AWG = protocols.AmneziaWG{PrivateKey: a.PrivateKey, PublicKey: a.PublicKey, Params: a.Params}
	return e, protocols.Account{Name: "alice", Password: "pw", WGPrivateKey: priv, WGPublicKey: pub, WGAddress: "10.30.0.2/32"}
}

func startAWGServer(t *testing.T) *awgFixture {
	t.Helper()
	fakeTunOpener(t)
	e, user := awgEnv(t)
	run := &fakeRunner{}
	port := freeUDP(t)
	srv, err := Start("amneziawg", Config{Env: e, Port: port, Users: []protocols.Account{user}, Runner: run})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	return &awgFixture{srv: srv, port: port, env: e, user: user, run: run}
}

// pingAWG connects a client built from the registry's profile (with tweak
// applied) and reports whether an echo to the server's tunnel address is
// answered within wait.
func (f *awgFixture) pingAWG(t *testing.T, wait time.Duration, tweak func(*protocols.Account, *protocols.Env)) bool {
	t.Helper()
	user, e := f.user, f.env
	if tweak != nil {
		tweak(&user, &e)
	}
	proto, _ := protocols.Get("amneziawg")
	p, err := proto.Profile(e, f.port, user)
	if err != nil {
		t.Fatal(err)
	}
	// The same device configuration the Android engine applies.
	uapi, err := awg.UAPI(p, netip.MustParseAddrPort("127.0.0.1:"+strconv.Itoa(f.port)))
	if err != nil {
		t.Fatal(err)
	}
	tn := newFakeTun(false)
	dev := device.NewDevice(tn, conn.NewDefaultBind(), &device.Logger{Verbosef: func(string, ...any) {}, Errorf: func(string, ...any) {}})
	defer dev.Close()
	if err := dev.IpcSet(uapi); err != nil {
		t.Fatal(err)
	}
	if err := dev.Up(); err != nil {
		t.Fatal(err)
	}
	src := net.ParseIP(strings.TrimSuffix(p.WireGuard.Address[0], "/32"))
	gateway := net.ParseIP("10.31.0.1")
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		select {
		case tn.in <- echoRequest(src, gateway, 9):
		default:
		}
		select {
		case r := <-tn.out:
			return len(r) >= 28 && r[20] == 0 && net.IP(r[12:16]).Equal(gateway)
		case <-time.After(300 * time.Millisecond):
		}
	}
	return false
}

func TestAmneziaWGServerAndGeneratedProfile(t *testing.T) {
	f := startAWGServer(t)
	if f.srv.Pool() != protocols.AWGPool {
		t.Fatalf("pool = %s", f.srv.Pool())
	}
	cmds := strings.Join(f.run.ran, "\n")
	for _, want := range []string{"ip addr add 10.31.0.1/16 dev omnitun-awg", "ip link set dev omnitun-awg up"} {
		if !strings.Contains(cmds, want) {
			t.Errorf("the interface was not set up: missing %q in %q", want, cmds)
		}
	}
	if !f.pingAWG(t, 10*time.Second, nil) {
		t.Fatal("the generated AmneziaWG profile did not carry a ping")
	}
}

func TestAmneziaWGServerRefusesStrangers(t *testing.T) {
	f := startAWGServer(t)
	if f.pingAWG(t, 2*time.Second, func(u *protocols.Account, _ *protocols.Env) {
		priv, pub, _ := store.NewWireGuardKeys()
		u.WGPrivateKey, u.WGPublicKey = priv, pub
	}) {
		t.Error("a client with an unknown key was served")
	}
	// Mismatched obfuscation parameters are not tested here: amneziawg-go keeps the
	// header values in package-level variables, so a client and a server in one
	// process always share them (a real server process runs a single device).
}

func TestAmneziaWGServerRefusesBadSetup(t *testing.T) {
	fakeTunOpener(t)
	e, user := awgEnv(t)

	if _, err := Start("amneziawg", Config{Env: e, Port: freeUDP(t), Users: []protocols.Account{{Name: "nokey"}}, Runner: &fakeRunner{}}); err == nil {
		t.Error("started with no user having a WireGuard key")
	}
	e2 := e
	e2.AWG.Params = map[string]string{"jc": "4"}
	if _, err := Start("amneziawg", Config{Env: e2, Port: freeUDP(t), Users: []protocols.Account{user}, Runner: &fakeRunner{}}); err == nil {
		t.Error("started with missing obfuscation parameters")
	}
	// A failing `ip` leaves no half-started device.
	run := &fakeRunner{failing: map[string]bool{"ip link set dev omnitun-awg up": true}}
	if _, err := Start("amneziawg", Config{Env: e, Port: freeUDP(t), Users: []protocols.Account{user}, Runner: run}); err == nil {
		t.Error("started although the interface could not be brought up")
	}
}

// The device's peer counters become per-user usage.
func TestAmneziaWGCountsTrafficPerUser(t *testing.T) {
	f := startAWGServer(t)
	m, ok := f.srv.(Meter)
	if !ok {
		t.Fatalf("%T does not count traffic per user", f.srv)
	}
	usage := func() Usage {
		for _, u := range m.Usage() {
			if u.User == "alice" {
				return u
			}
		}
		return Usage{}
	}
	if u := usage(); u.Up != 0 || u.Down != 0 {
		t.Fatalf("traffic before anyone connected: %+v", u)
	}
	if !f.pingAWG(t, 10*time.Second, nil) {
		t.Fatal("no ping through the tunnel")
	}
	u := usage()
	if u.Up < 28 || u.Down < 28 {
		t.Fatalf("usage after a ping: %+v", u)
	}
	for _, other := range m.Usage() {
		if other.User == "" {
			t.Fatal("a peer without an owner was reported")
		}
	}
}
