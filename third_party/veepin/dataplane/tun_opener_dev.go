//go:build !darwin

package dataplane

func wrapDevice(dev PacketDevice, name string) (*TUN, error) { return &TUN{dev: dev, name: name}, nil }

// devRead maps a closed device to ErrClosed, as the kernel TUN does.
func (t *TUN) devRead(buf []byte) (int, error) {
	n, err := t.dev.Read(buf)
	if err != nil && t.isClosed() {
		return 0, ErrClosed
	}
	return n, err
}

func (t *TUN) devWrite(pkt []byte) (int, error) {
	n, err := t.dev.Write(pkt)
	if err != nil && t.isClosed() {
		return 0, ErrClosed
	}
	return n, err
}
