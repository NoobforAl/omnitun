// Command awgclient is a Linux AmneziaWG client for the Docker end-to-end
// tests: it reads a client .conf (as the server's `link --file` prints it),
// creates a real kernel TUN device, applies the same device configuration
// the Android engine applies, and routes the given networks into it.
//
//	awgclient -conf amneziawg.conf -route 198.18.0.0/24
package main

import (
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"

	"github.com/NoobforAl/omnitun/core/awg"
	"github.com/NoobforAl/omnitun/core/importer"
)

func run(name string, args ...string) {
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		log.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, out)
	}
}

func main() {
	confPath := flag.String("conf", "", "client .conf")
	route := flag.String("route", "", "comma separated networks to send through the tunnel")
	name := flag.String("name", "omnitun-awgc", "interface name")
	flag.Parse()

	text, err := os.ReadFile(*confPath)
	if err != nil {
		log.Fatal(err)
	}
	res, err := importer.Import(string(text))
	if err != nil || len(res.Profiles) != 1 {
		log.Fatalf("import: %v %+v", err, res)
	}
	p := res.Profiles[0]
	endpoint, err := netip.ParseAddrPort(p.Server + ":" + strconv.Itoa(p.Port))
	if err != nil {
		log.Fatalf("endpoint %s:%d: %v", p.Server, p.Port, err)
	}
	uapi, err := awg.UAPI(p, endpoint)
	if err != nil {
		log.Fatal(err)
	}
	t, err := tun.CreateTUN(*name, 1280)
	if err != nil {
		log.Fatal(err)
	}
	dev := device.NewDevice(t, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "awgclient: "))
	if err := dev.IpcSet(uapi); err != nil {
		log.Fatal(err)
	}
	if err := dev.Up(); err != nil {
		log.Fatal(err)
	}
	run("ip", "addr", "add", p.WireGuard.Address[0], "dev", *name)
	run("ip", "link", "set", "dev", *name, "up")
	for _, r := range strings.Split(*route, ",") {
		if r = strings.TrimSpace(r); r != "" {
			run("ip", "route", "add", r, "dev", *name)
		}
	}
	fmt.Println("awgclient: up")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	dev.Close()
}
