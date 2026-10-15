//go:build linux

package awg

import (
	"github.com/amnezia-vpn/amneziawg-go/tun"
	"golang.org/x/sys/unix"
)

// createTUN wraps a duplicate of fd so the caller's descriptor stays valid.
func createTUN(fd int) (tun.Device, error) {
	dup, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	dev, _, err := tun.CreateUnmonitoredTUNFromFD(dup)
	if err != nil {
		unix.Close(dup)
		return nil, err
	}
	return dev, nil
}
