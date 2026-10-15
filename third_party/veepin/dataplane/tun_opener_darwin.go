package dataplane

import "errors"

// Supplied devices aren't supported on macOS (Omnitun patch).
func wrapDevice(dev PacketDevice, name string) (*TUN, error) {
	dev.Close()
	return nil, errors.New("dataplane: supplied TUN devices are not supported on macOS")
}
