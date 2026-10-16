package main

import (
	"fmt"
	"os"
	goruntime "runtime"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/install"
	"github.com/NoobforAl/omnitun/server/manager"
)

// installOptions are the flags the install, uninstall and firewall commands share.
type installFlags struct {
	user, binDir, firewall string
	dryRun                 bool
}

func (f *installFlags) add(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.user, "user", "omnitun", `user the service runs as ("root" to run as root)`)
	cmd.Flags().StringVar(&f.binDir, "bin-dir", "/usr/local/bin", "where the binary is installed")
	cmd.Flags().StringVar(&f.firewall, "firewall", "auto", "firewall manager: auto, ufw, firewalld or none")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "only print what would be done")
}

// options turns the flags and the server's configuration into install options.
func (f *installFlags) options(o *options, needServer bool) (install.Options, error) {
	opts := install.Options{BinDir: f.binDir, DataDir: o.dataDir, Firewall: f.firewall}
	if f.user != "root" {
		opts.User = f.user
	}
	switch f.firewall {
	case "auto", "ufw", "firewalld", "none":
	default:
		return opts, fmt.Errorf("%w: --firewall must be auto, ufw, firewalld or none", manager.ErrInvalid)
	}
	exe, err := os.Executable()
	if err != nil {
		return opts, err
	}
	opts.Binary = exe
	rules, err := o.manager().FirewallPorts()
	if err != nil {
		if needServer {
			return opts, err
		}
	}
	for _, r := range rules {
		opts.Ports = append(opts.Ports, install.Port{Port: r.Port, Proto: r.Proto, Purpose: r.Purpose})
	}
	return opts, nil
}

// run shows the plan, or runs it as root on Linux.
func (f *installFlags) run(plan install.Plan, runner install.Runner) error {
	if f.dryRun {
		install.Describe(plan, os.Stdout)
		return nil
	}
	if goruntime.GOOS != "linux" {
		return fmt.Errorf("installing as a service is supported on Linux only (add --dry-run to see the steps)")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("run this as root (sudo omnitun-server ...), or add --dry-run to see the steps")
	}
	if err := install.Execute(plan, runner, os.Stdout); err != nil {
		return err
	}
	for _, n := range plan.Notes {
		fmt.Printf("\n%s\n", n)
	}
	return nil
}

func newInstallCmd(o *options) *cobra.Command {
	var f installFlags
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install as a systemd service and open the firewall (Linux, run as root)",
		Long: `Sets the server up as a service in one go:

  - creates a system user and locks the data directory down to it,
  - copies this binary to --bin-dir,
  - writes a hardened systemd unit and starts it now and at every boot,
  - opens the ports of the enabled protocols in ufw or firewalld.

Run "omnitun-server init" first. Add --dry-run to see every step without
changing anything. If you change protocols or ports later, run
"omnitun-server firewall" to open the new ones.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := f.options(o, true)
			if err != nil {
				return err
			}
			runner := install.ExecRunner{}
			plan, err := install.BuildPlan(opts, install.Detect(runner))
			if err != nil {
				return err
			}
			return f.run(plan, runner)
		},
	}
	f.add(cmd)
	return cmd
}

func newUninstallCmd(o *options) *cobra.Command {
	var f installFlags
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove the service and its firewall rules (keeps the data)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := f.options(o, false)
			if err != nil {
				return err
			}
			runner := install.ExecRunner{}
			return f.run(install.BuildRemovalPlan(opts, install.Detect(runner)), runner)
		},
	}
	f.add(cmd)
	return cmd
}

func newFirewallCmd(o *options) *cobra.Command {
	var f installFlags
	cmd := &cobra.Command{
		Use:   "firewall",
		Short: "Open the ports the enabled protocols need in ufw or firewalld",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := f.options(o, true)
			if err != nil {
				return err
			}
			runner := install.ExecRunner{}
			return f.run(install.FirewallPlan(opts, install.Detect(runner)), runner)
		},
	}
	f.add(cmd)
	cmd.Flags().Lookup("user").Hidden = true
	cmd.Flags().Lookup("bin-dir").Hidden = true
	return cmd
}
