//go:build !linux

package awg

import (
	"errors"

	"github.com/amnezia-vpn/amneziawg-go/tun"
)

// createTUN is only implemented for Android/Linux, where the OS hands us a
// TUN file descriptor. Desktop AmneziaWG comes with the Windows phase.
func createTUN(int) (tun.Device, error) {
	return nil, errors.New("AmneziaWG needs Android VPN mode on this build")
}
