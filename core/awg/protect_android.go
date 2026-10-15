//go:build android

package awg

import "github.com/amnezia-vpn/amneziawg-go/conn"

// protectBind excludes the WireGuard UDP sockets from the VPN, as the
// official WireGuard Android app does.
func protectBind(bind conn.Bind, protect func(fd int) error) error {
	std, ok := bind.(*conn.StdNetBind)
	if !ok || protect == nil {
		return nil
	}
	if fd, err := std.PeekLookAtSocketFd4(); err == nil {
		if err := protect(fd); err != nil {
			return err
		}
	}
	if fd, err := std.PeekLookAtSocketFd6(); err == nil {
		if err := protect(fd); err != nil {
			return err
		}
	}
	return nil
}
