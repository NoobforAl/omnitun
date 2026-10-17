package mobile

import (
	"context"
	"net"
	"sync/atomic"
	"syscall"
)

// fallbackDNS answers when the network names no resolver of its own.
const fallbackDNS = "1.1.1.1"

// installResolver replaces Go's default resolver for the whole process. Go reads
// /etc/resolv.conf, which Android does not have, so it would ask [::1]:53 and every host
// name (AmneziaWG, SSTP and IKEv2 servers, the connectivity check, ...) would fail with
// "connection refused". This one asks the resolvers of the network the phone is online
// through, over sockets that bypass the VPN.
func installResolver(w *platformWrapper) {
	var next atomic.Uint32
	dialer := net.Dialer{
		Control: func(_, _ string, c syscall.RawConn) error {
			var protectErr error
			if err := c.Control(func(fd uintptr) { protectErr = w.p.Protect(int32(fd)) }); err != nil {
				return err
			}
			return protectErr
		},
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			servers := w.SystemDNS()
			if len(servers) == 0 {
				servers = []string{fallbackDNS}
			}
			// Go retries by dialing again, so rotating makes the retry reach another resolver.
			server := servers[int(next.Add(1)-1)%len(servers)]
			return dialer.DialContext(ctx, network, net.JoinHostPort(server, "53"))
		},
	}
}
