// Command sstpdev is a throwaway SSTP server for testing the client: its
// virtual interface answers pings, so a client can check the whole data path
// (ping the printed gateway through the tunnel) without a real network behind it.
//
//	go run ./tools/sstpdev -port 14443        (then: sstp://vpn:vpn@127.0.0.1:14443)
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"flag"
	"log"
	"math/big"
	"os"
	"os/signal"
	"time"

	"github.com/xen0bit/veepin/dataplane"
	"github.com/xen0bit/veepin/sstp"
)

type pingDevice struct {
	replies chan []byte
	closed  chan struct{}
}

func (d *pingDevice) Read(b []byte) (int, error) {
	select {
	case p := <-d.replies:
		return copy(b, p), nil
	case <-d.closed:
		return 0, errors.New("closed")
	}
}

func (d *pingDevice) Write(p []byte) (int, error) {
	if len(p) >= 28 && p[0]>>4 == 4 && p[9] == 1 && p[20] == 8 {
		r := append([]byte(nil), p...)
		copy(r[12:16], p[16:20])
		copy(r[16:20], p[12:16])
		r[10], r[11], r[20], r[22], r[23] = 0, 0, 0, 0, 0
		binary.BigEndian.PutUint16(r[10:], checksum(r[:20]))
		binary.BigEndian.PutUint16(r[22:], checksum(r[20:]))
		log.Printf("ping %d.%d.%d.%d -> %d.%d.%d.%d", p[12], p[13], p[14], p[15], p[16], p[17], p[18], p[19])
		d.replies <- r
	}
	return len(p), nil
}

func (d *pingDevice) Close() error { close(d.closed); return nil }

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

func main() {
	port := flag.Int("port", 14443, "TCP port")
	flag.Parse()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "sstpdev"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour)}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	keyDER, _ := x509.MarshalECPrivateKey(k)
	dataplane.TUNOpener = func(dataplane.TUNRequest) (dataplane.PacketDevice, error) {
		return &pingDevice{replies: make(chan []byte, 64), closed: make(chan struct{})}, nil
	}
	srv, err := sstp.NewServer(sstp.ServerConfig{
		Cert:     pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		Key:      pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
		ListenIP: "0.0.0.0", ListenPort: *port, Pool: "10.30.0.0/24",
		Users: map[string]string{"vpn": "vpn"},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("SSTP on :%d (user vpn / vpn); ping %s through the tunnel", *port, srv.Gateway())
	go func() { log.Fatal(srv.ListenAndServe()) }()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	srv.Close()
}
