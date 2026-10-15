//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"net"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// The engine must start and stop cleanly in proxy mode for every protocol.
// No traffic is sent, so unreachable example servers are fine.
func TestStartStopProxyMode(t *testing.T) {
	var profiles []*profile.Profile
	for _, link := range testLinks() {
		p, err := importer.ParseLink(link)
		if err != nil {
			t.Fatal(err)
		}
		profiles = append(profiles, p)
	}
	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			var states []Status
			e := New(func(s State) { states = append(states, s.Status) })
			port := freePort(t)
			if err := e.Start(p.AsSlice(), singbox.Options{ListenPort: port, LogLevel: "error"}); err != nil {
				t.Fatal(err)
			}
			conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
			if err != nil {
				t.Fatalf("proxy port not listening: %v", err)
			}
			conn.Close()
			if err := e.Stop(); err != nil {
				t.Fatal(err)
			}
			want := []Status{StatusStarting, StatusRunning, StatusStopped}
			if len(states) != len(want) {
				t.Fatalf("states = %v, want %v", states, want)
			}
			for i := range want {
				if states[i] != want[i] {
					t.Fatalf("states = %v, want %v", states, want)
				}
			}
		})
	}
	// All profiles together form one selector/urltest group.
	e := New(nil)
	if err := e.Start(profiles, singbox.Options{ListenPort: freePort(t), LogLevel: "error"}); err != nil {
		t.Fatal(err)
	}
	if err := e.Stop(); err != nil {
		t.Fatal(err)
	}
}
