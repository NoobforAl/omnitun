package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/qrterm"
)

func expiry(u manager.UserInfo) string {
	if u.ExpiresAt == nil {
		return "never"
	}
	return u.ExpiresAt.Local().Format("2006-01-02")
}

func quota(u manager.UserInfo) string {
	if u.QuotaGB <= 0 {
		return "unlimited"
	}
	return strconv.FormatFloat(u.QuotaGB, 'f', -1, 64) + " GB"
}

func printUsers(users []manager.UserInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tEXPIRES\tQUOTA\tCREATED\tNOTE")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", u.Name, u.Status, expiry(u), quota(u), u.CreatedAt.Format("2006-01-02"), u.Note)
	}
	w.Flush()
}

func newUserCmd(o *options) *cobra.Command {
	cmd := &cobra.Command{Use: "user", Short: "Manage users"}
	oneUser := func(u *manager.UserInfo, verb string) func() {
		return func() { fmt.Printf("user %s: %s (%s)\n", u.Name, verb, u.Status) }
	}

	var nu manager.NewUser
	add := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a user (credentials for every protocol, one subscription link)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nu.Name = args[0]
			m := o.manager()
			u, err := m.AddUser(nu)
			if err != nil {
				return err
			}
			set, err := m.Links(u.Name)
			if err != nil {
				return err
			}
			return o.print(map[string]any{"user": u, "links": set}, func() {
				fmt.Printf("Added %s.\n\n", u.Name)
				printLinkSet(set, false, false)
			})
		},
	}
	add.Flags().IntVar(&nu.Days, "days", 0, "account lifetime in days (0 = never expires)")
	add.Flags().Float64Var(&nu.QuotaGB, "quota", 0, "data cap in GB reported to clients (0 = unlimited)")
	add.Flags().StringVar(&nu.Note, "note", "", "free-form note")

	list := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			users, err := o.manager().Users()
			if err != nil {
				return err
			}
			return o.print(users, func() { printUsers(users) })
		},
	}

	del := &cobra.Command{
		Use:     "del <name>",
		Aliases: []string{"delete", "rm"},
		Short:   "Delete a user",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.manager().DeleteUser(args[0]); err != nil {
				return err
			}
			return o.print(map[string]string{"deleted": args[0]}, func() { fmt.Printf("user %s: deleted\n", args[0]) })
		},
	}

	toggle := func(use string, on bool) *cobra.Command {
		return &cobra.Command{
			Use:   use + " <name>",
			Short: strings.ToUpper(use[:1]) + use[1:] + " a user",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				u, err := o.manager().UpdateUser(args[0], manager.UserPatch{Enabled: &on})
				if err != nil {
					return err
				}
				return o.print(u, oneUser(u, use+"d"))
			},
		}
	}

	expire := &cobra.Command{
		Use:   "expire <name> <days|never>",
		Short: "Set the expiry to N days from now, or remove it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			days := 0
			if args[1] != "never" {
				n, err := strconv.Atoi(args[1])
				if err != nil || n < 1 {
					return fmt.Errorf("%w: days must be a positive number or \"never\", got %q", manager.ErrInvalid, args[1])
				}
				days = n
			}
			u, err := o.manager().UpdateUser(args[0], manager.UserPatch{ExpireDays: &days})
			if err != nil {
				return err
			}
			return o.print(u, func() { fmt.Printf("user %s: expires %s\n", u.Name, expiry(*u)) })
		},
	}

	quotaCmd := &cobra.Command{
		Use:   "quota <name> <GB|0>",
		Short: "Set the data cap in GB (0 = unlimited)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			gb, err := strconv.ParseFloat(args[1], 64)
			if err != nil || gb < 0 {
				return fmt.Errorf("%w: invalid quota %q", manager.ErrInvalid, args[1])
			}
			u, err := o.manager().UpdateUser(args[0], manager.UserPatch{QuotaGB: &gb})
			if err != nil {
				return err
			}
			return o.print(u, func() { fmt.Printf("user %s: quota %s\n", u.Name, quota(*u)) })
		},
	}

	note := &cobra.Command{
		Use:   "note <name> <text>",
		Short: "Set a user's note",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args[1:], " ")
			u, err := o.manager().UpdateUser(args[0], manager.UserPatch{Note: &text})
			if err != nil {
				return err
			}
			return o.print(u, oneUser(u, "note updated"))
		},
	}

	reset := &cobra.Command{
		Use:   "reset <name>",
		Short: "Issue new credentials (old links stop working)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u, err := o.manager().ResetUser(args[0])
			if err != nil {
				return err
			}
			return o.print(u, oneUser(u, "new credentials issued"))
		},
	}

	cmd.AddCommand(add, list, del, toggle("enable", true), toggle("disable", false), expire, quotaCmd, note, reset)
	return cmd
}

func newLinkCmd(o *options) *cobra.Command {
	var qr, subOnly bool
	var ovpn string
	cmd := &cobra.Command{
		Use:   "link <name>",
		Short: "Show a user's share links, subscription URL and QR codes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := o.manager()
			if ovpn != "" {
				text, err := m.OVPN(args[0], ovpn)
				if err != nil {
					return err
				}
				fmt.Print(text)
				return nil
			}
			set, err := m.Links(args[0])
			if err != nil {
				return err
			}
			return o.print(set, func() { printLinkSet(set, qr, subOnly) })
		},
	}
	cmd.Flags().BoolVar(&qr, "qr", false, "show QR codes")
	cmd.Flags().BoolVar(&subOnly, "sub", false, "only the subscription URL")
	cmd.Flags().StringVar(&ovpn, "ovpn", "", "print the user's OpenVPN config file: udp or tcp")
	return cmd
}

func printLinkSet(set *manager.LinkSet, qr, subOnly bool) {
	show := func(title, link string) {
		fmt.Printf("%s\n%s\n", title, link)
		if qr {
			if code, err := qrterm.Render(link); err == nil {
				fmt.Println(code)
			}
		}
		fmt.Println()
	}
	if set.Subscription != "" {
		show("Subscription (all protocols, auto-updating — recommended):", set.Subscription)
	}
	if subOnly {
		return
	}
	for _, l := range set.Links {
		show(l.Protocol+":", l.URL)
	}
	for _, proto := range []string{"openvpn-udp", "openvpn-tcp"} {
		url, on := set.OVPN[proto]
		switch {
		case !on:
		case url != "":
			show(proto+" (.ovpn file for any OpenVPN app):", url)
		default:
			fmt.Printf("%s: link %s --ovpn %s > %s.ovpn\n\n", proto, set.User, strings.TrimPrefix(proto, "openvpn-"), proto)
		}
	}
}
