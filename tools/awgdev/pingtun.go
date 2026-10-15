package main

import (
	"encoding/binary"
	"log"
	"os"

	"github.com/amnezia-vpn/amneziawg-go/tun"
)

// pingTun is a fake TUN device that answers ICMP echo requests to any address.
type pingTun struct {
	replies chan []byte
	events  chan tun.Event
	closed  chan struct{}
}

func newPingTun() *pingTun {
	t := &pingTun{replies: make(chan []byte, 64), events: make(chan tun.Event, 1), closed: make(chan struct{})}
	t.events <- tun.EventUp
	return t
}

func (t *pingTun) File() *os.File           { return nil }
func (t *pingTun) MTU() (int, error)        { return 1280, nil }
func (t *pingTun) Name() (string, error)    { return "pingtun", nil }
func (t *pingTun) Events() <-chan tun.Event { return t.events }
func (t *pingTun) BatchSize() int           { return 1 }

func (t *pingTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	select {
	case p := <-t.replies:
		sizes[0] = copy(bufs[0][offset:], p)
		return 1, nil
	case <-t.closed:
		return 0, os.ErrClosed
	}
}

// Write receives packets from the peer and queues echo replies.
func (t *pingTun) Write(bufs [][]byte, offset int) (int, error) {
	for _, b := range bufs {
		p := b[offset:]
		if len(p) < 28 || p[0]>>4 != 4 || p[9] != 1 {
			continue // not IPv4 ICMP
		}
		ihl := int(p[0]&0x0f) * 4
		if len(p) < ihl+8 || p[ihl] != 8 {
			continue // not an echo request
		}
		r := append([]byte(nil), p...)
		copy(r[12:16], p[16:20]) // swap addresses
		copy(r[16:20], p[12:16])
		r[8] = 64 // TTL
		r[10], r[11] = 0, 0
		binary.BigEndian.PutUint16(r[10:], checksum(r[:ihl]))
		r[ihl] = 0 // echo reply
		r[ihl+2], r[ihl+3] = 0, 0
		binary.BigEndian.PutUint16(r[ihl+2:], checksum(r[ihl:]))
		log.Printf("ping from %d.%d.%d.%d", p[12], p[13], p[14], p[15])
		select {
		case t.replies <- r:
		default:
		}
	}
	return len(bufs), nil
}

func (t *pingTun) Close() error {
	close(t.closed)
	close(t.events)
	return nil
}

func checksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(b); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(b[i:]))
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}
	return ^uint16(sum)
}
