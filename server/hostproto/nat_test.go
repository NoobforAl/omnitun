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
	for _, want := range []string{"table ip omnitun_hosts", "10.10.10.0/24, 10.9.0.0/24", "masquerade", "hook forward", "hook postrouting"} {
		if !strings.Contains(string(rules), want) {
			t.Errorf("rules lack %q:\n%s", want, rules)
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
	want := []string{
		"iptables -t nat -A POSTROUTING -s 10.9.0.0/24 -j MASQUERADE",
		"iptables -I FORWARD -s 10.9.0.0/24 -j ACCEPT",
		"iptables -I FORWARD -d 10.9.0.0/24 -j ACCEPT",
	}
	if strings.Join(r.ran, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commands:\n%s", strings.Join(r.ran, "\n"))
	}
	r.ran = nil
	undo()
	removed := strings.Join(r.ran, "\n")
	for _, w := range []string{"iptables -t nat -D POSTROUTING -s 10.9.0.0/24 -j MASQUERADE", "iptables -D FORWARD -s 10.9.0.0/24 -j ACCEPT", "iptables -D FORWARD -d 10.9.0.0/24 -j ACCEPT"} {
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
	// iptables: the second rule fails, so the first is taken back.
	r = &fakeRunner{have: map[string]bool{"iptables": true}, failing: map[string]bool{"iptables -I FORWARD -s 10.9.0.0/24 -j ACCEPT": true}}
	if _, err := EnableNAT(r, tempDir(t), []string{"10.9.0.0/24"}); err == nil {
		t.Fatal("a failing iptables was ignored")
	}
	if last := r.ran[len(r.ran)-1]; !strings.Contains(last, "-D POSTROUTING") {
		t.Fatalf("the masquerade rule was not removed after the failure: %v", r.ran)
	}
}
