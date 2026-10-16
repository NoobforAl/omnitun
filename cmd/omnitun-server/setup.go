package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
)

func newStatusCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the server summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := o.manager().Server()
			if err != nil {
				return err
			}
			return o.print(info, func() {
				fmt.Printf("%s (%s)\n", info.Name, info.PublicHost)
				fmt.Printf("users: %d (%d active)   protocols enabled: %d\n", info.Users, info.ActiveUsers, info.Protocols)
				if info.Running {
					fmt.Printf("server: running, %d connection(s) now\n", info.Connections)
				} else {
					fmt.Println("server: not running (start it with: omnitun-server run)")
				}
				if info.SubscriptionPort > 0 {
					fmt.Printf("subscriptions on port %d\n", info.SubscriptionPort)
				}
			})
		},
	}
}

func printProtocols(list []manager.ProtocolInfo, subPort int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROTOCOL\tNAME\tPORT\tTRANSPORT\tSTATUS")
	for _, p := range list {
		status := "disabled"
		if p.Enabled {
			status = "enabled"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", p.Key, p.Name, p.Port, p.Transport, status)
	}
	if subPort > 0 {
		fmt.Fprintf(w, "subscriptions\t\t%d\ttcp\thttp\n", subPort)
	}
	w.Flush()
}

func newProtoCmd(o *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proto",
		Short: "List protocols, switch them on or off, change ports",
	}
	keys := strings.Join(store.AllProtocols, ", ")

	list := &cobra.Command{
		Use:   "list",
		Short: "Protocols, ports and status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := o.manager()
			protos, err := m.Protocols()
			if err != nil {
				return err
			}
			info, _ := m.Server()
			return o.print(protos, func() { printProtocols(protos, info.SubscriptionPort) })
		},
	}
	setEnabled := func(use string, on bool) *cobra.Command {
		return &cobra.Command{
			Use:       use + " <protocol>",
			Short:     strings.ToUpper(use[:1]) + use[1:] + " a protocol (" + keys + ")",
			Args:      cobra.ExactArgs(1),
			ValidArgs: store.AllProtocols,
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := o.manager().SetProtocol(args[0], manager.ProtocolPatch{Enabled: &on})
				if err != nil {
					return err
				}
				return o.print(p, func() { fmt.Printf("%s: enabled=%v port=%d\n", p.Key, p.Enabled, p.Port) })
			},
		}
	}
	port := &cobra.Command{
		Use:       "port <protocol> <port>",
		Short:     "Change a protocol's port",
		Args:      cobra.ExactArgs(2),
		ValidArgs: store.AllProtocols,
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("%w: port %q is not a number", manager.ErrInvalid, args[1])
			}
			p, err := o.manager().SetProtocol(args[0], manager.ProtocolPatch{Port: &n})
			if err != nil {
				return err
			}
			return o.print(p, func() { fmt.Printf("%s: enabled=%v port=%d\n", p.Key, p.Enabled, p.Port) })
		},
	}
	cmd.AddCommand(list, setEnabled("enable", true), setEnabled("disable", false), port)
	return cmd
}
