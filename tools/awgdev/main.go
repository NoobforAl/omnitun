// Command awgdev is a throwaway AmneziaWG server for testing the client: it
// runs entirely in userspace and answers pings to 10.8.0.1 inside the tunnel,
// which proves the handshake and the obfuscated data path. It prints a client
// .conf to import.
//
// Example (on an Android phone, since adb can't forward UDP):
//
//	adb push awgdev /data/local/tmp/ && adb shell /data/local/tmp/awgdev -endpoint 127.0.0.1
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"golang.org/x/crypto/curve25519"
)

// Obfuscation parameters; client and server must match.
const obfs = "jc=4\njmin=40\njmax=70\ns1=15\ns2=20\nh1=1234567\nh2=2345678\nh3=3456789\nh4=4567890\n"

func keyPair() (priv, pub []byte) {
	priv = make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		log.Fatal(err)
	}
	priv[0] &= 248
	priv[31] = (priv[31] & 127) | 64
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		log.Fatal(err)
	}
	return priv, pub
}

func main() {
	port := flag.Int("port", 51820, "UDP listen port")
	endpoint := flag.String("endpoint", "127.0.0.1", "host the client should connect to")
	flag.Parse()

	srvPriv, srvPub := keyPair()
	cliPriv, cliPub := keyPair()
	b64 := base64.StdEncoding.EncodeToString

	dev := device.NewDevice(newPingTun(), conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "awgdev: "))
	uapi := fmt.Sprintf("private_key=%s\nlisten_port=%d\n%spublic_key=%s\nallowed_ip=10.8.0.2/32\n",
		hex.EncodeToString(srvPriv), *port, obfs, hex.EncodeToString(cliPub))
	if err := dev.IpcSet(uapi); err != nil {
		log.Fatal(err)
	}
	if err := dev.Up(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(`[Interface]
PrivateKey = %s
Address = 10.8.0.2/32
DNS = 1.1.1.1
Jc = 4
Jmin = 40
Jmax = 70
S1 = 15
S2 = 20
H1 = 1234567
H2 = 2345678
H3 = 3456789
H4 = 4567890

[Peer]
PublicKey = %s
AllowedIPs = 10.8.0.0/24
Endpoint = %s:%d
`, b64(cliPriv), b64(srvPub), *endpoint, *port)
	log.Printf("listening on udp :%d; ping 10.8.0.1 through the tunnel", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	dev.Close()
}
