// Command omnitun is a developer CLI for the core engine: import configs,
// print the generated sing-box config, or run a tunnel.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

const usage = `usage: omnitun <command> [flags] <file|->

commands:
  import   parse links / subscription / .conf and print profiles as JSON
  config   print the sing-box config generated for the profiles
  run      start the tunnel (proxy mode by default) until Ctrl+C

flags (config, run):
  -tun          capture all traffic with a TUN device (needs admin/root)
  -port N       local SOCKS5/HTTP port in proxy mode (default 2080)
  -select ID    profile ID to use (default: auto, lowest latency)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	tun := fs.Bool("tun", false, "")
	port := fs.Int("port", 2080, "")
	sel := fs.String("select", "", "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	text, err := readInput(fs.Arg(0))
	if err != nil {
		return err
	}
	res, err := importer.Import(text)
	if err != nil {
		return err
	}
	for _, e := range res.Errors {
		fmt.Fprintln(os.Stderr, "skipped:", e)
	}

	opts := singbox.Options{ListenPort: *port, Selected: *sel, Mode: singbox.ModeProxy}
	if *tun {
		opts.Mode = singbox.ModeTUN
	}

	switch cmd {
	case "import":
		return printJSON(res.Profiles)
	case "config":
		cfg, skipped, err := singbox.Build(res.Profiles, opts)
		for _, s := range skipped {
			fmt.Fprintln(os.Stderr, "skipped:", s)
		}
		if err != nil {
			return err
		}
		if err := engine.CheckConfig(cfg); err != nil {
			return fmt.Errorf("generated config is invalid: %w", err)
		}
		return printJSON(cfg)
	case "run":
		eng := engine.New(func(s engine.State) {
			fmt.Fprintf(os.Stderr, "state: %s %s\n", s.Status, s.Error)
			for _, sk := range s.Skipped {
				fmt.Fprintln(os.Stderr, "skipped:", sk)
			}
		})
		if err := eng.Start(res.Profiles, opts); err != nil {
			return err
		}
		if opts.Mode == singbox.ModeProxy {
			fmt.Fprintf(os.Stderr, "proxy listening on 127.0.0.1:%d (SOCKS5 + HTTP)\n", *port)
		}
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		return eng.Stop()
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	return nil
}

func readInput(path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	if _, err := os.Stat(path); err != nil {
		// Allow passing a single link directly.
		return path, nil
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
