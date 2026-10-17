package mobile

import (
	"context"
	"net"
	"testing"
)

func TestStartTunnelFromJSONNeedsTheCore(t *testing.T) {
	mu.Lock()
	saved := srv
	srv = nil
	mu.Unlock()
	defer func() {
		mu.Lock()
		srv = saved
		mu.Unlock()
	}()
	if err := StartTunnelFromJSON(`{"profiles":[]}`); err == nil {
		t.Fatal("started without a running core")
	}
}

// fakePlatform answers only Interfaces; the embedded nil interface covers the rest.
type fakePlatform struct {
	Platform
	interfaces string
	protected  *int
}

func (f fakePlatform) Protect(int32) error {
	if f.protected != nil {
		*f.protected++
	}
	return nil
}

func (f fakePlatform) Interfaces() (string, error) { return f.interfaces, nil }

// Go cannot read a phone's resolvers itself, so they come from the platform's network list:
// the usable ones of real, up networks, IPv4 first, never loopback or link-local.
func TestSystemDNSFromThePlatformNetworks(t *testing.T) {
	w := newPlatformWrapper(fakePlatform{interfaces: `[
		{"name":"lo","index":1,"flags":73,"type":"other","dns":["127.0.0.1"]},
		{"name":"tun0","index":9,"flags":65,"type":"other","dns":["10.0.0.2"]},
		{"name":"wlan0","index":3,"flags":65,"type":"wifi","dns":["fe80::1","2001:db8::53","5.200.200.200","0.0.0.0","8.8.8.8"]}
	]`})
	got := w.SystemDNS()
	want := []string{"5.200.200.200", "8.8.8.8", "2001:db8::53"}
	if len(got) != len(want) {
		t.Fatalf("SystemDNS = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SystemDNS = %v, want %v", got, want)
		}
	}
}

// Go's own resolver would ask [::1]:53 on Android. The installed one asks the network's
// resolver, over a socket the VPN does not capture.
func TestInstalledResolverUsesProtectedSockets(t *testing.T) {
	saved := net.DefaultResolver
	defer func() { net.DefaultResolver = saved }()
	protected := 0
	installResolver(newPlatformWrapper(fakePlatform{
		interfaces: `[{"name":"wlan0","index":3,"flags":65,"type":"wifi","dns":["192.0.2.53"]}]`,
		protected:  &protected,
	}))
	conn, err := net.DefaultResolver.Dial(context.Background(), "udp", "[::1]:53")
	if err == nil {
		defer conn.Close()
		if got := conn.RemoteAddr().String(); got != "192.0.2.53:53" {
			t.Errorf("dialed %s, want the network's resolver 192.0.2.53:53", got)
		}
	}
	if protected != 1 {
		t.Errorf("the DNS socket was protected %d times, want 1", protected)
	}
}
