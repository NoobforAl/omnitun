package runtime

import (
	"net/netip"
	"sync"

	ovpn "github.com/sagernet/sing-openvpn"
)

// ovpnUsers remembers which user holds which OpenVPN tunnel address. sing-box's
// OpenVPN server hands out addresses from a pool and does not say who got one;
// the patched sing-openvpn (third_party/sing-openvpn) reports each assignment.
type ovpnUsers struct {
	mu sync.RWMutex
	m  map[netip.Addr]string
}

func (o *ovpnUsers) user(a netip.Addr) string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.m[a]
}

func (o *ovpnUsers) observe(e ovpn.SessionEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.m == nil {
		o.m = map[netip.Addr]string{}
	}
	for _, a := range []netip.Addr{e.IPv4, e.IPv6} {
		if !a.IsValid() {
			continue
		}
		if e.Up {
			o.m[a.Unmap()] = e.Username
		} else {
			delete(o.m, a.Unmap())
		}
	}
}

// watchOpenVPN starts following OpenVPN sessions; the observer is process-wide,
// so the newest server wins and stopWatchingOpenVPN lets go of it.
func (s *Server) watchOpenVPN() { ovpn.SessionObserver = s.ovpn.observe }

func (s *Server) stopWatchingOpenVPN() { ovpn.SessionObserver = nil }
