package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/manager"
)

func newInitCmd(o *options) *cobra.Command {
	var (
		opts      manager.InitOptions
		enabled   []string
		ports     map[string]int
		firstUser string
		wizard    bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new server (guided in a terminal, or with flags)",
		Long: `Create a new server: keys, certificate and the protocols you choose.

Run it without --host in a terminal for a step-by-step wizard, or give
everything as flags:

  omnitun-server init --host vpn.example.com --protocols vless-reality,trojan,hysteria2 \
      --port trojan=9443 --first-user alice

Protocols: ` + strings.Join(protocols.Keys(), ", ") + `.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Enabled, opts.Ports = enabled, ports
			interactive := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) && !o.json
			if opts.PublicHost == "" || wizard {
				if !interactive {
					return fmt.Errorf("%w: --host is required (run in a terminal for the wizard)", manager.ErrInvalid)
				}
				w, err := runWizard(opts)
				if err != nil {
					return err
				}
				opts, firstUser = w.opts, w.firstUser
			}
			m := o.manager()
			info, err := m.Init(opts)
			if err != nil {
				return err
			}
			var added *manager.LinkSet
			if firstUser != "" {
				if _, err := m.AddUser(manager.NewUser{Name: firstUser}); err != nil {
					return err
				}
				if added, err = m.Links(firstUser); err != nil {
					return err
				}
			}
			protos, _ := m.Protocols()
			return o.print(map[string]any{"server": info, "protocols": protos, "first_user": added}, func() {
				fmt.Printf("Initialized %q in %s\n\n", info.Name, o.dataDir)
				printProtocols(protos, info.SubscriptionPort)
				if added != nil {
					fmt.Printf("\nUser %s:\n\n", firstUser)
					printLinkSet(added, false, true)
				}
				fmt.Println("\nNext: omnitun-server user add <name>   then   omnitun-server run")
				fmt.Println("Open the ports above in your firewall (see the TRANSPORT column).")
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.PublicHost, "host", "", "public IP or domain clients connect to (omit in a terminal for the wizard)")
	f.StringVar(&opts.Name, "name", "Omnitun", "server name shown in clients")
	f.StringVar(&opts.RealityHandshake, "reality-site", "www.microsoft.com:443", "site VLESS+Reality and ShadowTLS imitate")
	f.IntVar(&opts.SubPort, "sub-port", 2096, "subscription endpoint port")
	f.StringSliceVar(&enabled, "protocols", nil, "protocols to enable, comma separated (default: all)")
	f.StringToIntVar(&ports, "port", nil, "port for a protocol, e.g. --port trojan=9443 (repeatable)")
	f.StringVar(&firstUser, "first-user", "", "also create this user and print the subscription link")
	f.BoolVar(&wizard, "wizard", false, "ask the questions even when --host is given")
	return cmd
}

// wizardResult is what the wizard collected.
type wizardResult struct {
	opts      manager.InitOptions
	firstUser string
}

// detectPublicIP asks a public service for this machine's address; it
// returns "" when offline or on any failure (the wizard then asks).
func detectPublicIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	if ip := net.ParseIP(strings.TrimSpace(string(body))); ip != nil {
		return ip.String()
	}
	return ""
}

func portValidator(s string) error {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 65535 {
		return errors.New("enter a port between 1 and 65535")
	}
	return nil
}

// runWizard asks the questions of `init` in a terminal.
func runWizard(start manager.InitOptions) (*wizardResult, error) {
	host, name, site, subPort := start.PublicHost, start.Name, start.RealityHandshake, strconv.Itoa(start.SubPort)
	if host == "" {
		host = detectPublicIP()
	}
	if name == "" {
		name = "Omnitun"
	}
	if site == "" {
		site = "www.microsoft.com:443"
	}
	if start.SubPort == 0 {
		subPort = "2096"
	}

	var chosen []string
	options := make([]huh.Option[string], 0, len(protocols.Specs()))
	for _, s := range protocols.Specs() {
		label := fmt.Sprintf("%-26s %-7s %s", s.Name, s.Transport, s.Description)
		options = append(options, huh.NewOption(label, s.Key).Selected(len(start.Enabled) == 0 || contains(start.Enabled, s.Key)))
	}

	first := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Omnitun server setup").
				Description("A few questions; every answer can be changed later with the CLI, the TUI or the web API."),
			huh.NewInput().Title("Public address").Description("The IP or domain your users connect to").Value(&host).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("an address is required")
					}
					return nil
				}),
			huh.NewInput().Title("Server name").Description("Shown in your users' apps").Value(&name),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().Title("Protocols").
				Description("Space to toggle. More protocols = more ways in on restrictive networks; you can switch them later.").
				Options(options...).Value(&chosen).
				Validate(func(v []string) error {
					if len(v) == 0 {
						return errors.New("choose at least one protocol")
					}
					return nil
				}),
		),
	)
	if err := first.Run(); err != nil {
		return nil, err
	}

	// Per-protocol details for what was chosen.
	portStrings := map[string]*string{}
	var fields []huh.Field
	needSite := false
	for _, key := range chosen {
		p, _ := protocols.Get(key)
		s := p.Spec()
		def := strconv.Itoa(s.DefaultPort)
		if v, ok := start.Ports[key]; ok {
			def = strconv.Itoa(v)
		}
		portStrings[key] = &def
		fields = append(fields, huh.NewInput().Title(s.Name+" port").Description(s.Transport).Value(&def).Validate(portValidator))
		if key == "vless-reality" || key == "shadowtls" {
			needSite = true
		}
	}
	if needSite {
		fields = append([]huh.Field{huh.NewInput().Title("Site to imitate").
			Description("Real HTTPS site for VLESS+Reality and ShadowTLS (host:port)").Value(&site)}, fields...)
	}
	var firstUser string
	fields = append(fields,
		huh.NewInput().Title("Subscription port").Description("Where users' apps fetch their links").Value(&subPort).Validate(portValidator),
		huh.NewInput().Title("First user (optional)").Description("Creates the user and prints the link at the end").Value(&firstUser),
	)
	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return nil, err
	}

	res := &wizardResult{firstUser: strings.TrimSpace(firstUser)}
	res.opts = manager.InitOptions{
		PublicHost: strings.TrimSpace(host), Name: strings.TrimSpace(name), RealityHandshake: strings.TrimSpace(site),
		Enabled: chosen, Ports: map[string]int{},
	}
	res.opts.SubPort, _ = strconv.Atoi(strings.TrimSpace(subPort))
	for key, v := range portStrings {
		res.opts.Ports[key], _ = strconv.Atoi(strings.TrimSpace(*v))
	}
	return res, nil
}

func contains(list []string, s string) bool { return slices.Contains(list, s) }
