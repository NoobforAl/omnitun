package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
)

// run executes the CLI against dataDir and returns its stdout.
func run(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := newRootCmd()
	root.SetArgs(append([]string{"--data", dataDir}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	err := root.Execute()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestCLIFlow(t *testing.T) {
	dir := t.TempDir()
	if _, err := run(t, dir, "init"); err == nil {
		t.Fatal("init without --host succeeded")
	}
	if out, err := run(t, dir, "init", "--host", "203.0.113.7"); err != nil || !strings.Contains(out, "vless-reality") {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if _, err := run(t, dir, "init", "--host", "203.0.113.7"); exitCode(err) != 4 {
		t.Fatalf("second init: %v (code %d), want conflict", err, exitCode(err))
	}

	out, err := run(t, dir, "--json", "user", "add", "alice", "--days", "5", "--quota", "10")
	if err != nil {
		t.Fatal(err)
	}
	var added struct {
		User  manager.UserInfo `json:"user"`
		Links manager.LinkSet  `json:"links"`
	}
	if err := json.Unmarshal([]byte(out), &added); err != nil || added.User.Name != "alice" || len(added.Links.Links) == 0 {
		t.Fatalf("add output: %v\n%s", err, out)
	}

	if _, err := run(t, dir, "user", "add", "alice"); exitCode(err) != 4 {
		t.Fatalf("duplicate user code = %d", exitCode(err))
	}
	if _, err := run(t, dir, "user", "disable", "ghost"); exitCode(err) != 3 {
		t.Fatalf("missing user code = %d", exitCode(err))
	}
	if _, err := run(t, dir, "user", "quota", "alice", "lots"); exitCode(err) != 2 {
		t.Fatalf("bad quota code = %d", exitCode(err))
	}

	run(t, dir, "user", "disable", "alice")
	out, _ = run(t, dir, "--json", "user", "list")
	var users []manager.UserInfo
	json.Unmarshal([]byte(out), &users)
	if len(users) != 1 || users[0].Status != "disabled" {
		t.Fatalf("users = %+v", users)
	}

	run(t, dir, "proto", "port", "trojan", "9443")
	run(t, dir, "proto", "disable", "tuic")
	out, _ = run(t, dir, "--json", "proto", "list")
	var protos []manager.ProtocolInfo
	json.Unmarshal([]byte(out), &protos)
	for _, p := range protos {
		if p.Key == "trojan" && p.Port != 9443 || p.Key == "tuic" && p.Enabled {
			t.Fatalf("protocol %+v", p)
		}
	}
	if _, err := run(t, dir, "proto", "enable", "nope"); exitCode(err) != 3 {
		t.Fatalf("unknown protocol code = %d", exitCode(err))
	}

	if out, err := run(t, dir, "link", "alice", "--sub"); err != nil || !strings.Contains(out, "http://203.0.113.7:2096/") {
		t.Fatalf("link --sub: %v\n%s", err, out)
	}
	if out, err := run(t, dir, "link", "alice", "--ovpn", "udp"); err != nil || !strings.Contains(out, "remote 203.0.113.7") {
		t.Fatalf("link --ovpn: %v\n%s", err, out)
	}
	if _, err := run(t, dir, "user", "del", "alice"); err != nil {
		t.Fatal(err)
	}
}

func TestAPICommands(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "--host", "203.0.113.7")
	out, _ := run(t, dir, "--json", "api", "token")
	var a store.API
	json.Unmarshal([]byte(out), &a)
	if a.Enabled || a.Token == "" {
		t.Fatalf("defaults = %+v", a)
	}
	run(t, dir, "api", "enable")
	run(t, dir, "api", "listen", "0.0.0.0", "9100")
	out, _ = run(t, dir, "--json", "api", "token", "--new")
	var b store.API
	json.Unmarshal([]byte(out), &b)
	if !b.Enabled || b.Listen != "0.0.0.0" || b.Port != 9100 || b.Token == a.Token {
		t.Fatalf("after changes = %+v", b)
	}
}

func TestLiveTokenFollowsTheState(t *testing.T) {
	old := tokenTTL
	tokenTTL = 20 * time.Millisecond
	defer func() { tokenTTL = old }()

	dir := t.TempDir()
	run(t, dir, "init", "--host", "203.0.113.7")
	run(t, dir, "api", "enable")
	m := manager.New(store.Open(dir))
	tok := liveToken(m)
	first := tok()
	if first == "" {
		t.Fatal("no token while enabled")
	}
	if a, _ := m.SetAPI(manager.APIPatch{NewToken: true}); a.Token == first {
		t.Fatal("token not rotated")
	}
	time.Sleep(50 * time.Millisecond)
	if tok() == first {
		t.Fatal("running server still honours the rotated-out token")
	}
	off := false
	m.SetAPI(manager.APIPatch{Enabled: &off})
	time.Sleep(50 * time.Millisecond)
	if tok() != "" {
		t.Fatal("disabled API still has a token")
	}
}
