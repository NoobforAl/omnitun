package runtime

import (
	"net/netip"
	"testing"

	ovpn "github.com/sagernet/sing-openvpn"
)

func TestOpenVPNUsersFollowSessions(t *testing.T) {
	var o ovpnUsers
	a := netip.MustParseAddr("10.8.0.2")
	b := netip.MustParseAddr("10.8.0.3")
	o.observe(ovpn.SessionEvent{Username: "alice", IPv4: a, Up: true})
	o.observe(ovpn.SessionEvent{Username: "bob", IPv4: b, Up: true})
	if o.user(a) != "alice" || o.user(b) != "bob" {
		t.Fatalf("got %q %q", o.user(a), o.user(b))
	}
	// An address freed and handed to someone else belongs to the new holder.
	o.observe(ovpn.SessionEvent{Username: "alice", IPv4: a})
	if o.user(a) != "" {
		t.Fatalf("released address still owned by %q", o.user(a))
	}
	o.observe(ovpn.SessionEvent{Username: "bob", IPv4: a, Up: true})
	if o.user(a) != "bob" {
		t.Fatalf("got %q", o.user(a))
	}
}

func TestTunnelUserOfConnections(t *testing.T) {
	s := &Server{}
	s.ovpn.observe(ovpn.SessionEvent{Username: "carol", IPv4: netip.MustParseAddr("10.8.0.9"), Up: true})
	s.wgUsers = map[netip.Addr]string{netip.MustParseAddr("10.20.0.2"): "dave"}
	for addr, want := range map[string]string{"10.8.0.9": "carol", "10.20.0.2": "dave", "10.8.0.77": ""} {
		if got := s.userByTunnelIP(netip.MustParseAddr(addr)); got != want {
			t.Errorf("%s: got %q, want %q", addr, got, want)
		}
	}
	for key, want := range map[string]bool{"wireguard": true, "openvpn-udp": true, "openvpn-tcp": true, "trojan": false} {
		if tunnelProtocol(key) != want {
			t.Errorf("tunnelProtocol(%q) = %v", key, !want)
		}
	}
}
