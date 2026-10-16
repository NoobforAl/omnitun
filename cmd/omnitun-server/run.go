package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/api"
	"github.com/NoobforAl/omnitun/server/runtime"
	"github.com/NoobforAl/omnitun/server/store"
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
				handler := api.New(api.Options{Manager: m, Token: a.Token, Reload: srv.Reload})
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := api.Serve(ctx, a.Listen, a.Port, handler, logger.Printf); err != nil {
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
			fmt.Printf("web API: %s, listening on %s:%d\n", state, a.Listen, a.Port)
			fmt.Printf("token:   %s\n", a.Token)
			fmt.Printf("docs:    http://%s:%d/docs\n", a.Listen, a.Port)
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
			Use: use, Short: "Turn the web API " + map[bool]string{true: "on", false: "off"}[on] + " (takes effect on the next `run`)", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return apply(func() (store.API, error) { return o.manager().SetAPI(apiPatchEnabled(on)) })
			},
		}
	}
	token := &cobra.Command{
		Use: "token", Short: "Print the API token; --new issues a fresh one", Args: cobra.NoArgs,
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
	cmd.AddCommand(status, enable("enable", true), enable("disable", false), token, listen)
	return cmd
}

func newServiceCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "service",
		Short: "Print a systemd unit for this server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			dir, _ := filepath.Abs(o.dataDir)
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
		},
	}
}
