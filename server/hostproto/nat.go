package hostproto

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Runner executes a command; tests replace it.
type Runner interface {
	Run(argv ...string) error
	// Has reports whether a command exists on this host.
	Has(name string) bool
}

// ExecRunner runs real commands.
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(argv ...string) error {
	out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(argv, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Has implements Runner.
func (ExecRunner) Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// natTable is the nftables table the server owns; it is removed on stop.
const natTable = "omnitun_hosts"

// nftRules is the ruleset for the client pools: masquerade what leaves the
// host and accept forwarding to and from the pools.
func nftRules(pools []string) string {
	sorted := append([]string(nil), pools...)
	sort.Strings(sorted)
	set := strings.Join(sorted, ", ")
	return fmt.Sprintf(`table ip %[1]s {
	chain forward {
		type filter hook forward priority -1; policy accept;
		ip saddr { %[2]s } accept
		ip daddr { %[2]s } accept
	}
	chain postrouting {
		type nat hook postrouting priority 100; policy accept;
		ip saddr { %[2]s } masquerade
	}
}
`, natTable, set)
}

// EnableNAT lets clients of the given pools reach the internet through this
// host and returns a function that removes the rules again. It uses nftables
// and falls back to iptables. dir is where the nft ruleset file is written.
func EnableNAT(r Runner, dir string, pools []string) (undo func(), err error) {
	if len(pools) == 0 {
		return func() {}, nil
	}
	switch {
	case r.Has("nft"):
		file := filepath.Join(dir, "nat.nft")
		if err := os.WriteFile(file, []byte(nftRules(pools)), 0o600); err != nil {
			return nil, err
		}
		// Start clean if a crashed run left the table behind.
		r.Run("nft", "delete", "table", "ip", natTable)
		if err := r.Run("nft", "-f", file); err != nil {
			os.Remove(file)
			return nil, err
		}
		return func() {
			r.Run("nft", "delete", "table", "ip", natTable)
			os.Remove(file)
		}, nil
	case r.Has("iptables"):
		var added [][]string
		undo := func() {
			for _, rule := range added {
				r.Run(append([]string{"iptables"}, rule...)...)
			}
		}
		for _, p := range pools {
			for _, rule := range [][]string{
				{"-t", "nat", "-A", "POSTROUTING", "-s", p, "-j", "MASQUERADE"},
				{"-I", "FORWARD", "-s", p, "-j", "ACCEPT"},
				{"-I", "FORWARD", "-d", p, "-j", "ACCEPT"},
			} {
				if err := r.Run(append([]string{"iptables"}, rule...)...); err != nil {
					undo()
					return nil, err
				}
				// The same rule with -D removes it.
				del := append([]string(nil), rule...)
				for i, a := range del {
					if a == "-A" || a == "-I" {
						del[i] = "-D"
					}
				}
				added = append([][]string{del}, added...)
			}
		}
		return undo, nil
	}
	return nil, fmt.Errorf("neither nft nor iptables is installed, so clients could not reach the internet")
}

// ForwardingEnabled reports whether the kernel forwards IPv4 packets.
func ForwardingEnabled() bool {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	return err == nil && strings.TrimSpace(string(data)) == "1"
}
