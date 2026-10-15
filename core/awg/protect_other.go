//go:build !android

package awg

import "github.com/amnezia-vpn/amneziawg-go/conn"

// protectBind is only needed on Android, where the app's own sockets would
// otherwise be routed into its VPN.
func protectBind(conn.Bind, func(fd int) error) error { return nil }
