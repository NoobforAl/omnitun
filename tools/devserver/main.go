// Command devserver runs a local Shadowsocks server for manual testing, e.g.
// with a phone over USB:  adb reverse tcp:8388 tcp:8388  then import
// ss://aes-128-gcm:omnitun-dev@127.0.0.1:8388#dev  on the phone.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

func main() {
	listen := flag.String("listen", "127.0.0.1", "listen address")
	port := flag.Int("port", 8388, "listen port")
	password := flag.String("password", "omnitun-dev", "shadowsocks password")
	logLevel := flag.String("log", "info", "log level")
	flag.Parse()

	cfg := map[string]any{
		"log": map[string]any{"level": *logLevel, "timestamp": true},
		"inbounds": []any{map[string]any{
			"type": "shadowsocks", "tag": "ss-in", "listen": *listen, "listen_port": *port,
			"method": "aes-128-gcm", "password": *password,
		}},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
	}
	raw, _ := json.Marshal(cfg)
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	opts, err := sjson.UnmarshalExtendedContext[option.Options](ctx, raw)
	if err != nil {
		log.Fatal(err)
	}
	server, err := box.New(box.Options{Context: ctx, Options: opts})
	if err != nil {
		log.Fatal(err)
	}
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	fmt.Printf("shadowsocks listening on %s:%d\nlink: ss://aes-128-gcm:%s@127.0.0.1:%d#dev\n", *listen, *port, *password, *port)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
