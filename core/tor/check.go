package tor

import (
	"context"
	"errors"
	"net"
	"time"
)

// CheckCircuit opens a connection through the SOCKS5 port to a Tor-friendly
// host and returns once Tor has built a circuit for it (the connection is
// closed straight away). It proves Tor can carry traffic, which a finished
// bootstrap doesn't: the guards may still be unreachable.
func CheckCircuit(ctx context.Context, socksAddr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", socksAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	} else {
		conn.SetDeadline(time.Now().Add(time.Minute))
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		return err
	}
	greeting := make([]byte, 2)
	if _, err := readFull(conn, greeting); err != nil {
		return err
	}
	const host = "check.torproject.org"
	req := append([]byte{5, 1, 0, 3, byte(len(host))}, host...)
	req = append(req, 0x01, 0xbb) // port 443
	if _, err := conn.Write(req); err != nil {
		return err
	}
	reply := make([]byte, 10)
	if _, err := readFull(conn, reply); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if reply[1] != 0 {
		return errors.New("Tor could not build a circuit")
	}
	return nil
}

func readFull(c net.Conn, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		m, err := c.Read(b[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
