package hostproto

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	have    map[string]bool
	failing map[string]bool
	ran     []string
}

func (f *fakeRunner) Run(argv ...string) error {
	line := strings.Join(argv, " ")
	f.ran = append(f.ran, line)
	if f.failing[line] {
		return errors.New("failed")
	}
	return nil
}

func (f *fakeRunner) Has(name string) bool { return f.have[name] }

func ruleLines(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		out = append(out, strings.TrimSpace(l))
	}
	return out
}

func indexOfLine(lines []string, prefix string) int {
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return i
		}
	}
	return -1
}

func TestEnableNATWithNftables(t *testing.T) {
	dir := tempDir(t)
	r := &fakeRunner{have: map[string]bool{"nft": true}}
	undo, err := EnableNAT(r, dir, []string{"10.9.0.0/24", "10.10.10.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "nat.nft")
	rules, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	text := string(rules)
	for _, want := range []string{"table ip omnitun_hosts", "masquerade", "hook forward", "hook postrouting",
		"ct state established,related ip daddr { 10.10.10.0/24, 10.9.0.0/24 } accept"} {
		if !strings.Contains(text, want) {
			t.Errorf("rules lack %q:\n%s", want, text)
		}
	}
	// The forwarding policy, in order: replies in, no client-to-client or
	// private-network access, then the internet, and nothing else towards clients.
	lines := ruleLines(text)
	pools := "{ 10.10.10.0/24, 10.9.0.0/24 }"
	order := []string{"ct state established,related", "ip saddr " + pools + " ip daddr {", "ip saddr " + pools + " accept", "ip daddr " + pools + " drop"}
	last := -1
	for _, want := range order {
		at := indexOfLine(lines, want)
		if at < 0 || at <= last {
			t.Fatalf("rule %q missing or out of order in:\n%s", want, text)
		}
		last = at
	}
	drop := lines[indexOfLine(lines, "ip saddr "+pools+" ip daddr {")]
	for _, private := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "100.64.0.0/10", "127.0.0.0/8", "10.9.0.0/24"} {
		if !strings.Contains(drop, private) || !strings.HasSuffix(drop, "drop") {
			t.Errorf("the drop rule does not cover %s: %s", private, drop)
		}
	}
	if len(r.ran) != 2 || r.ran[0] != "nft delete table ip omnitun_hosts" || r.ran[1] != "nft -f "+file {
		t.Fatalf("commands = %v", r.ran)
	}
	undo()
	if last := r.ran[len(r.ran)-1]; last != "nft delete table ip omnitun_hosts" {
		t.Fatalf("undo ran %q", last)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("the ruleset file was left behind")
	}
}

func TestEnableNATFallsBackToIptablesAndUndoesIt(t *testing.T) {
	r := &fakeRunner{have: map[string]bool{"iptables": true}}
	undo, err := EnableNAT(r, tempDir(t), []string{"10.9.0.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ran[0] != "iptables -t nat -A POSTROUTING -s 10.9.0.0/24 -j MASQUERADE" {
		t.Fatalf("first command: %q", r.ran[0])
	}
	// Rules are inserted last-to-first at position 1, so they end up in policy
	// order: the last insert is the established/related accept.
	inserts := r.ran[1:]
	if got := inserts[len(inserts)-1]; got != "iptables -I FORWARD 1 -d 10.9.0.0/24 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT" {
		t.Errorf("the topmost rule is %q", got)
	}
	if got := inserts[0]; got != "iptables -I FORWARD 1 -d 10.9.0.0/24 -j DROP" {
		t.Errorf("the bottom rule is %q", got)
	}
	all := strings.Join(inserts, "\n")
	for _, want := range []string{"-s 10.9.0.0/24 -j ACCEPT", "-s 10.9.0.0/24 -d 10.0.0.0/8 -j DROP", "-s 10.9.0.0/24 -d 192.168.0.0/16 -j DROP", "-s 10.9.0.0/24 -d 10.9.0.0/24 -j DROP"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing a rule for %q:\n%s", want, all)
		}
	}
	n := len(r.ran)
	r.ran = nil
	undo()
	if len(r.ran) != n {
		t.Fatalf("undo ran %d commands for %d rules", len(r.ran), n)
	}
	removed := strings.Join(r.ran, "\n")
	for _, w := range []string{"iptables -t nat -D POSTROUTING -s 10.9.0.0/24 -j MASQUERADE", "iptables -D FORWARD -s 10.9.0.0/24 -j ACCEPT", "iptables -D FORWARD -d 10.9.0.0/24 -j DROP"} {
		if !strings.Contains(removed, w) {
			t.Errorf("undo lacks %q:\n%s", w, removed)
		}
	}
}

func TestEnableNATFailures(t *testing.T) {
	if _, err := EnableNAT(&fakeRunner{}, tempDir(t), []string{"10.9.0.0/24"}); err == nil || !strings.Contains(err.Error(), "neither nft nor iptables") {
		t.Fatalf("no tools: %v", err)
	}
	if undo, err := EnableNAT(&fakeRunner{}, tempDir(t), nil); err != nil || undo == nil {
		t.Fatalf("no pools: %v", err)
	}
	// A rule that fails leaves nothing behind.
	dir := tempDir(t)
	r := &fakeRunner{have: map[string]bool{"nft": true}, failing: map[string]bool{"nft -f " + filepath.Join(dir, "nat.nft"): true}}
	if _, err := EnableNAT(r, dir, []string{"10.9.0.0/24"}); err == nil {
		t.Fatal("a failing nft was ignored")
	}
	if _, err := os.Stat(filepath.Join(dir, "nat.nft")); !os.IsNotExist(err) {
		t.Error("the ruleset file was kept after a failure")
	}
	// iptables: a rule fails part-way, so everything added before it is taken back.
	r = &fakeRunner{have: map[string]bool{"iptables": true}, failing: map[string]bool{"iptables -I FORWARD 1 -s 10.9.0.0/24 -j ACCEPT": true}}
	if _, err := EnableNAT(r, tempDir(t), []string{"10.9.0.0/24"}); err == nil {
		t.Fatal("a failing iptables was ignored")
	}
	if !strings.Contains(strings.Join(r.ran, "\n"), "-D POSTROUTING") || !strings.Contains(strings.Join(r.ran, "\n"), "-D FORWARD -d 10.9.0.0/24 -j DROP") {
		t.Fatalf("earlier rules were not removed after the failure: %v", r.ran)
	}
}
