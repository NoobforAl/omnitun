// Command omnitun-server runs a multi-protocol VPN server and manages its
// users from the command line, a terminal UI (tui) or a web API (api).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
)

// options are the flags every command shares.
type options struct {
	dataDir string
	json    bool
}

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

// manager opens the data directory.
func (o *options) manager() *manager.Manager {
	return manager.New(store.Open(o.dataDir))
}

// print writes v as JSON when --json is set and otherwise calls text.
func (o *options) print(v any, text func()) error {
	if o.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	text()
	return nil
}

func newRootCmd() *cobra.Command {
	o := &options{}
	root := &cobra.Command{
		Use:   "omnitun-server",
		Short: "Multi-protocol VPN server with user management",
		Long: `omnitun-server runs VLESS+Reality, Shadowsocks-2022, Trojan, Hysteria2, TUIC,
AnyTLS, ShadowTLS, NaiveProxy and OpenVPN from one process, with a user
database, one subscription link per user, a terminal UI and a web API.

Quick start:
  omnitun-server init --host YOUR_IP_OR_DOMAIN
  omnitun-server user add alice
  omnitun-server run

The data directory (keys, users) defaults to /var/lib/omnitun-server as root,
otherwise ~/.omnitun-server; override with --data or OMNITUN_SERVER_DATA.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&o.dataDir, "data", defaultDataDir(), "data directory")
	root.PersistentFlags().BoolVar(&o.json, "json", false, "print results as JSON")
	root.AddCommand(
		newInitCmd(o), newStatusCmd(o), newProtoCmd(o), newUserCmd(o), newLinkCmd(o),
		newAPICmd(o), newBackupCmd(o), newRestoreCmd(o), newImportCmd(o), newCertCmd(o), newLogsCmd(o), newInstallCmd(o), newUninstallCmd(o), newFirewallCmd(o), newRunCmd(o), newTUICmd(o), newServiceCmd(o),
	)
	return root
}

// exitCode maps manager errors to process exit codes.
func exitCode(err error) int {
	switch {
	case errors.Is(err, manager.ErrInvalid):
		return 2
	case errors.Is(err, manager.ErrNotFound):
		return 3
	case errors.Is(err, manager.ErrConflict):
		return 4
	}
	return 1
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}
