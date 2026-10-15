package dataplane

// Omnitun patch: lets an embedding app supply the TUN device (Android's
// VpnService file descriptor, or a test device) instead of /dev/net/tun.
// The protocol calls OpenTUNFor after its handshake, so the request carries
// the negotiated addresses an app needs to create the interface.

import (
	"fmt"
	"net"
)

// PacketDevice moves whole IP packets, like a TUN file descriptor.
type PacketDevice interface {
	Read(buf []byte) (int, error)
	Write(pkt []byte) (int, error)
	Close() error
}

// TUNRequest describes the interface a protocol is about to open.
type TUNRequest struct {
	Name    string
	IPv4    net.IP
	Netmask net.IP
	IPv6    net.IP
	Prefix6 int
	DNS     []net.IP
	MTU     int
	Server  bool // a server-side (responder) device
}

// TUNOpener, when set, supplies every TUN device. It is process-wide.
var TUNOpener func(TUNRequest) (PacketDevice, error)

// OpenTUNFor opens the interface for req through TUNOpener, or the kernel
// TUN driver when no opener is set.
func OpenTUNFor(req TUNRequest) (*TUN, error) {
	if TUNOpener == nil {
		return OpenTUNGSO(req.Name)
	}
	dev, err := TUNOpener(req)
	if err != nil {
		return nil, fmt.Errorf("dataplane: open TUN: %w", err)
	}
	return wrapDevice(dev, req.Name)
}
