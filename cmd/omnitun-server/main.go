// Command omnitun-server runs a multi-protocol VPN server (VLESS+Reality,
// Shadowsocks-2022, Trojan, Hysteria2, TUIC) and manages its users.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/NoobforAl/omnitun/server/qrterm"
	"github.com/NoobforAl/omnitun/server/runtime"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/tui"
)

const usage = `omnitun-server — multi-protocol VPN server

usage: omnitun-server [--data DIR] <command> [args]

setup
  init --host IP_OR_DOMAIN [--name NAME] [--reality-site HOST:PORT] [--sub-port PORT]
  proto list                          protocols, ports and status
  proto enable|disable <protocol>     protocols: ` + "vless-reality, shadowsocks, trojan, hysteria2, tuic" + `
  proto port <protocol> <port>

users
  user add <name> [--days N] [--quota GB] [--note TEXT]
  user list
  user del <name>
  user enable|disable <name>
  user expire <name> <days|never>
  user quota <name> <GB|0>
  user reset <name>                   new credentials (old links stop working)
  link <name> [--qr] [--sub]          share links; --sub: subscription URL only

run
  run                                 start the server (reloads when users change)
  tui                                 interactive user & config manager
  service                             print a systemd unit for this server

The data directory (keys, users) defaults to /var/lib/omnitun-server as root,
otherwise ~/.omnitun-server; override with --data or OMNITUN_SERVER_DATA.
`

func defaultDataDir() string {
	if d := os.Getenv("OMNITUN_SERVER_DATA"); d != "" {
		return d
	}
	if goruntime.GOOS == "linux" && os.Geteuid() == 0 {
		return "/var/lib/omnitun-server"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".omnitun-server"
	}
	return filepath.Join(home, ".omnitun-server")
}

func main() {
	global := flag.NewFlagSet("omnitun-server", flag.ExitOnError)
	data := global.String("data", defaultDataDir(), "data directory")
	global.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	_ = global.Parse(os.Args[1:])
	args := global.Args()
	if len(args) == 0 {
		global.Usage()
		os.Exit(2)
	}
	s := store.Open(*data)
	if err := dispatch(s, args[0], args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func dispatch(s *store.Store, cmd string, args []string) error {
	switch cmd {
	case "init":
		return cmdInit(s, args)
	case "proto":
		return cmdProto(s, args)
	case "user":
		return cmdUser(s, args)
	case "link":
		return cmdLink(s, args)
	case "run":
		return cmdRun(s)
	case "tui":
		return tui.Run(s)
	case "service":
		return cmdService(s)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	}
	return fmt.Errorf("unknown command %q (see: omnitun-server help)", cmd)
}

// parse splits positional arguments from flags, wherever they appear.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) > 0 {
			pos = append(pos, args[0])
			args = args[1:]
		}
	}
	return pos, nil
}

func cmdInit(s *store.Store, args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	host := fs.String("host", "", "public IP or domain clients connect to (required)")
	name := fs.String("name", "Omnitun", "server name shown in clients")
	site := fs.String("reality-site", "www.microsoft.com:443", "site VLESS+Reality impersonates")
	subPort := fs.Int("sub-port", 2096, "subscription endpoint port")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	st, err := store.Init(s, store.InitOptions{Name: *name, PublicHost: *host, RealityHandshake: *site, SubPort: *subPort})
	if err != nil {
		return err
	}
	fmt.Printf("Initialized %q in %s\n\n", st.Server.Name, s.Dir)
	printProtocols(st)
	fmt.Println("\nNext: omnitun-server user add <name>   then   omnitun-server run")
	fmt.Println("Open the ports above in your firewall (TCP for vless-reality/shadowsocks/trojan, UDP for hysteria2/tuic/shadowsocks).")
	return nil
}

func printProtocols(st *store.State) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROTOCOL\tPORT\tTRANSPORT\tSTATUS")
	for _, name := range store.AllProtocols {
		p := st.Server.Protocols[name]
		status := "disabled"
		if p.Enabled {
			status = "enabled"
		}
		transport := "tcp"
		switch name {
		case store.Hysteria2, store.TUIC:
			transport = "udp"
		case store.Shadowsocks:
			transport = "tcp+udp"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", name, p.Port, transport, status)
	}
	if sub := st.Server.Subscription; sub.Enabled {
		fmt.Fprintf(w, "subscriptions\t%d\ttcp\thttp\n", sub.Port)
	}
	w.Flush()
}

func cmdProto(s *store.Store, args []string) error {
	if len(args) == 0 || args[0] == "list" {
		st, err := s.Load()
		if err != nil {
			return err
		}
		printProtocols(st)
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: proto enable|disable|port <protocol> [port]")
	}
	name := args[1]
	if !slices.Contains(store.AllProtocols, name) {
		return fmt.Errorf("unknown protocol %q (have: %s)", name, strings.Join(store.AllProtocols, ", "))
	}
	return s.Update(func(st *store.State) error {
		p := st.Server.Protocols[name]
		switch args[0] {
		case "enable":
			p.Enabled = true
		case "disable":
			p.Enabled = false
		case "port":
			if len(args) < 3 {
				return fmt.Errorf("usage: proto port <protocol> <port>")
			}
			port, err := strconv.Atoi(args[2])
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %q", args[2])
			}
			p.Port = port
		default:
			return fmt.Errorf("unknown proto command %q", args[0])
		}
		fmt.Printf("%s: enabled=%v port=%d\n", name, p.Enabled, p.Port)
		return nil
	})
}

func cmdUser(s *store.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: user add|list|del|enable|disable|expire|quota|reset ...")
	}
	sub, args := args[0], args[1:]
	if sub == "list" {
		st, err := s.Load()
		if err != nil {
			return err
		}
		printUsers(st)
		return nil
	}
	fs := flag.NewFlagSet("user "+sub, flag.ExitOnError)
	days := fs.Int("days", 0, "account lifetime in days (0 = never expires)")
	quota := fs.Float64("quota", 0, "data cap in GB reported to clients (0 = unlimited)")
	note := fs.String("note", "", "free-form note")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return fmt.Errorf("user %s: missing user name", sub)
	}
	name := pos[0]
	var created *store.User
	err = s.Update(func(st *store.State) error {
		if sub == "add" {
			u, err := store.AddUser(st, name)
			if err != nil {
				return err
			}
			if *days > 0 {
				t := time.Now().AddDate(0, 0, *days).UTC()
				u.ExpiresAt = &t
			}
			u.QuotaGB, u.Note = *quota, *note
			created = u
			return nil
		}
		u := st.User(name)
		if u == nil {
			return fmt.Errorf("no user %q", name)
		}
		switch sub {
		case "del", "delete", "rm":
			return store.DeleteUser(st, name)
		case "enable":
			u.Enabled = true
		case "disable":
			u.Enabled = false
		case "expire":
			if len(pos) < 2 {
				return fmt.Errorf("usage: user expire <name> <days|never>")
			}
			if pos[1] == "never" {
				u.ExpiresAt = nil
			} else {
				n, err := strconv.Atoi(pos[1])
				if err != nil {
					return fmt.Errorf("invalid days %q", pos[1])
				}
				t := time.Now().AddDate(0, 0, n).UTC()
				u.ExpiresAt = &t
			}
		case "quota":
			if len(pos) < 2 {
				return fmt.Errorf("usage: user quota <name> <GB|0>")
			}
			gb, err := strconv.ParseFloat(pos[1], 64)
			if err != nil || gb < 0 {
				return fmt.Errorf("invalid quota %q", pos[1])
			}
			u.QuotaGB = gb
		case "reset":
			store.ResetCredentials(u)
		default:
			return fmt.Errorf("unknown user command %q", sub)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if created != nil {
		st, _ := s.Load()
		fmt.Printf("Added %s.\n\n", created.Name)
		printLinks(st, st.User(created.Name), false, false)
		return nil
	}
	fmt.Printf("user %s: done\n", name)
	return nil
}

func printUsers(st *store.State) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tEXPIRES\tQUOTA\tCREATED\tNOTE")
	now := time.Now()
	for _, u := range st.Users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", u.Name, u.Status(now), expiry(u), quota(u), u.CreatedAt.Format("2006-01-02"), u.Note)
	}
	w.Flush()
}

func expiry(u *store.User) string {
	if u.ExpiresAt == nil {
		return "never"
	}
	return u.ExpiresAt.Local().Format("2006-01-02")
}

func quota(u *store.User) string {
	if u.QuotaGB <= 0 {
		return "unlimited"
	}
	return strconv.FormatFloat(u.QuotaGB, 'f', -1, 64) + " GB"
}

func cmdLink(s *store.Store, args []string) error {
	fs := flag.NewFlagSet("link", flag.ExitOnError)
	qr := fs.Bool("qr", false, "show QR codes")
	subOnly := fs.Bool("sub", false, "only the subscription URL")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return fmt.Errorf("usage: link <name> [--qr] [--sub]")
	}
	st, err := s.Load()
	if err != nil {
		return err
	}
	u := st.User(pos[0])
	if u == nil {
		return fmt.Errorf("no user %q", pos[0])
	}
	printLinks(st, u, *qr, *subOnly)
	return nil
}

func printLinks(st *store.State, u *store.User, qr, subOnly bool) {
	show := func(title, link string) {
		fmt.Printf("%s\n%s\n", title, link)
		if qr {
			if code, err := qrterm.Render(link); err == nil {
				fmt.Println(code)
			}
		}
		fmt.Println()
	}
	if sub := serverconf.SubscriptionURL(st, u); sub != "" {
		show("Subscription (all protocols, auto-updating — recommended):", sub)
	}
	if subOnly {
		return
	}
	for _, l := range serverconf.Links(st, u) {
		show(l.Protocol+":", l.URL)
	}
}

func cmdRun(s *store.Store) error {
	logger := log.New(os.Stderr, "omnitun-server: ", log.LstdFlags)
	srv := runtime.New(s, logger.Printf)
	if err := srv.Start(); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go srv.Watch(ctx)
	logger.Printf("running with data in %s; Ctrl+C to stop", s.Dir)
	<-ctx.Done()
	return srv.Close()
}

func cmdService(s *store.Store) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir, _ := filepath.Abs(s.Dir)
	fmt.Printf(`# Save as /etc/systemd/system/omnitun-server.service, then:
#   systemctl daemon-reload && systemctl enable --now omnitun-server
[Unit]
Description=Omnitun VPN server
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=%s --data %s run
Restart=on-failure
RestartSec=3
LimitNOFILE=1048576
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`, exe, dir)
	return nil
}
