//go:build with_acme && with_quic && with_utls && with_wireguard && with_gvisor

package runtime

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/classicvpn"
	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
)

// recordingRunner stands in for nft/iptables.
type recordingRunner struct {
	mu  sync.Mutex
	ran []string
}

func (r *recordingRunner) Run(argv ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ran = append(r.ran, strings.Join(argv, " "))
	return nil
}
func (r *recordingRunner) Has(name string) bool { return name == "nft" }
func (r *recordingRunner) commands() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.ran, "\n")
}

type nopDevice struct{ closed chan struct{} }

func (d nopDevice) Read([]byte) (int, error)    { <-d.closed; return 0, os.ErrClosed }
func (d nopDevice) Write(p []byte) (int, error) { return len(p), nil }
func (d nopDevice) Close() error                { return nil }

// Enabling SSTP starts its server and the NAT for its clients; disabling it
// stops both again.
func TestHostProtocolsFollowTheConfiguration(t *testing.T) {
	classicvpn.ServerOpener = func(classicvpn.Interface) (classicvpn.Device, error) {
		return nopDevice{make(chan struct{})}, nil
	}
	defer func() { classicvpn.ServerOpener = nil }()

	s, srv := setup(t)
	runner := &recordingRunner{}
	srv.runner = runner

	m := manager.New(s)
	if srv.hosts["sstp"] != nil {
		t.Fatal("SSTP runs although it is off by default")
	}
	port := freeTCP(t)
	on := true
	if _, err := m.SetProtocol(store.SSTP, manager.ProtocolPatch{Enabled: &on, Port: &port}); err != nil {
		t.Fatal(err)
	}
	if err := srv.Reload(); err != nil {
		t.Fatal(err)
	}
	if srv.hosts["sstp"] == nil {
		t.Fatalf("SSTP did not start; log: %v", nil)
	}
	waitFor(t, "SSTP to listen", 5*time.Second, func() bool {
		c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
		if err == nil {
			c.Close()
		}
		return err == nil
	})
	if cmds := runner.commands(); !strings.Contains(cmds, "nft -f") {
		t.Fatalf("NAT was not set up: %q", cmds)
	}

	off := false
	if _, err := m.SetProtocol(store.SSTP, manager.ProtocolPatch{Enabled: &off}); err != nil {
		t.Fatal(err)
	}
	if err := srv.Reload(); err != nil {
		t.Fatal(err)
	}
	if srv.hosts["sstp"] != nil {
		t.Fatal("SSTP still runs after being disabled")
	}
	if cmds := runner.commands(); !strings.HasSuffix(cmds, "nft delete table ip omnitun_hosts") {
		t.Fatalf("NAT rules were not removed: %q", cmds)
	}
}
