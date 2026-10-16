package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/api"
	"github.com/NoobforAl/omnitun/server/install"
	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/runtime"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/tlsfiles"
	"github.com/NoobforAl/omnitun/server/tui"
)

func newRunCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the server (reloads when users change); serves the web API if enabled",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := o.manager()
			logger := log.New(os.Stderr, "omnitun-server: ", log.LstdFlags)
			srv := runtime.New(m.Store, logger.Printf)
			if err := srv.Start(); err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() { defer wg.Done(); srv.Watch(ctx) }()

			if a, err := m.API(); err == nil && a.Enabled {
				handler := api.New(api.Options{Manager: m, Token: liveToken(m), Reload: srv.Reload})
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := api.Serve(ctx, a.Listen, a.Port, handler, apiTLS(m, a), logger.Printf); err != nil {
						logger.Printf("web API stopped: %v", err)
					}
				}()
			}
			logger.Printf("running with data in %s; Ctrl+C to stop", m.Store.Dir)
			<-ctx.Done()
			wg.Wait()
			return srv.Close()
		},
	}
}

// apiTLS is the HTTPS configuration of the web API (nil for plain HTTP): the
// server's certificate, the ACME one when there is one.
func apiTLS(m *manager.Manager, a store.API) *tls.Config {
	if !a.TLS {
		return nil
	}
	st, err := m.Store.Load()
	if err != nil {
		return nil
	}
	domain := ""
	if st.Server.ACMEActive() {
		domain = st.Server.ACME.Domain
	}
	return tlsfiles.New(serverconf.ACMEDir(m.Store), domain, m.Store.Path(st.Server.CertFile), m.Store.Path(st.Server.KeyFile)).TLSConfig()
}

// tokenTTL bounds how long a rotated or disabled token can still be accepted
// by a running server, whatever the file system's timestamp resolution.
var tokenTTL = 2 * time.Second

// liveToken returns the API's current token, re-read from the state file at
// least every tokenTTL (and sooner when its size or modification time change),
// so `api token --new` and `api disable` take effect on a running server
// without a restart.
func liveToken(m *manager.Manager) func() string {
	var (
		mu      sync.Mutex
		at      time.Time
		modTime time.Time
		size    int64
		token   string
	)
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		mt, sz := m.Store.ModTime(), m.Store.Size()
		if time.Since(at) < tokenTTL && mt.Equal(modTime) && sz == size {
			return token
		}
		at, modTime, size, token = time.Now(), mt, sz, ""
		if a, err := m.API(); err == nil && a.Enabled {
			token = a.Token
		}
		return token
	}
}

func newTUICmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive user and server manager",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return tui.Run(o.manager().Store) },
	}
}

func newAPICmd(o *options) *cobra.Command {
	cmd := &cobra.Command{Use: "api", Short: "Configure the web API (served by `run`)"}

	show := func(a store.API) func() {
		return func() {
			state := "disabled"
			if a.Enabled {
				state = "enabled"
			}
			scheme := "http"
			if a.TLS {
				scheme = "https"
			}
			fmt.Printf("web API: %s, listening on %s:%d (%s)\n", state, a.Listen, a.Port, scheme)
			fmt.Printf("token:   %s\n", a.Token)
			fmt.Printf("docs:    %s://%s:%d/docs\n", scheme, a.Listen, a.Port)
		}
	}
	apply := func(p func() (store.API, error)) error {
		a, err := p()
		if err != nil {
			return err
		}
		return o.print(a, show(a))
	}

	status := &cobra.Command{
		Use: "status", Short: "Show the web API settings and token", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error { return apply(o.manager().API) },
	}
	enable := func(use string, on bool) *cobra.Command {
		return &cobra.Command{
			Use: use, Short: "Turn the web API " + map[bool]string{true: "on (starts with the next `run`)", false: "off (a running server stops accepting its token at once)"}[on], Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return apply(func() (store.API, error) { return o.manager().SetAPI(apiPatchEnabled(on)) })
			},
		}
	}
	token := &cobra.Command{
		Use: "token", Short: "Print the API token; --new issues a fresh one (the old one stops working at once, even on a running server)", Args: cobra.NoArgs,
	}
	newToken := token.Flags().Bool("new", false, "issue a new token (the old one stops working)")
	token.RunE = func(cmd *cobra.Command, args []string) error {
		return apply(func() (store.API, error) {
			if *newToken {
				return o.manager().SetAPI(apiPatchToken())
			}
			return o.manager().API()
		})
	}
	listen := &cobra.Command{
		Use: "listen <host> [port]", Short: "Set the listen address, e.g. 127.0.0.1 or 0.0.0.0", Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			var port *int
			if len(args) == 2 {
				n, err := strconv.Atoi(args[1])
				if err != nil {
					return fmt.Errorf("invalid port %q", args[1])
				}
				port = &n
			}
			return apply(func() (store.API, error) { return o.manager().SetAPI(apiPatchListen(host, port)) })
		},
	}
	tlsCmd := &cobra.Command{
		Use: "tls <on|off>", Short: "Serve the web API over HTTPS with the server's certificate (applies on the next `run`)",
		Args: cobra.ExactArgs(1), ValidArgs: []string{"on", "off"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "on" && args[0] != "off" {
				return fmt.Errorf("%w: use on or off", manager.ErrInvalid)
			}
			on := args[0] == "on"
			return apply(func() (store.API, error) { return o.manager().SetAPI(manager.APIPatch{TLS: &on}) })
		},
	}
	cmd.AddCommand(status, enable("enable", true), enable("disable", false), token, listen, tlsCmd)
	return cmd
}

func newServiceCmd(o *options) *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Print the systemd unit for this server (`install` writes the same one)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			dir, _ := filepath.Abs(o.dataDir)
			if user == "root" {
				user = ""
			}
			fmt.Printf("# Save as %s, then:\n#   systemctl daemon-reload && systemctl enable --now %s\n", install.UnitPath, install.UnitName)
			fmt.Print(install.Unit(filepath.ToSlash(exe), filepath.ToSlash(dir), user))
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "root", "user the service runs as")
	return cmd
}
