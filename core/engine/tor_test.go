package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

const testFingerprint = "D9A82D2F9C2F65A18407B1D2B764F130847F8B5D"

func TestBridgeFingerprint(t *testing.T) {
	for line, want := range map[string]bool{
		"obfs4 192.0.2.1:443 " + testFingerprint + " cert=abc iat-mode=0":  true,
		"snowflake 192.0.2.3:80 fingerprint=" + testFingerprint + " url=x": true,
		"192.0.2.2:9001 " + testFingerprint:                                true,
		"meek_lite 192.0.2.20:80 url=https://x.example front=y utls=Hello": false,
		"obfs4 192.0.2.1:443 cert=abc iat-mode=0":                          false,
	} {
		if got := hasFingerprint(line); got != want {
			t.Errorf("hasFingerprint(%q) = %v, want %v", line, got, want)
		}
	}
	p := &profile.Profile{Protocol: profile.Tor, Tor: &profile.TorOpts{Bridges: []string{"meek_lite 192.0.2.20:80 url=x"}}}
	if _, err := torBridges(p); err == nil {
		t.Error("a bridge line without a fingerprint was accepted")
	}
}

func TestPlainError(t *testing.T) {
	for err, want := range map[error]string{
		context.DeadlineExceeded: "the server did not answer in time",
		errString("dial tcp 1.2.3.4:443: connect: connection refused"): "the server refused the connection",
		errString("something unusual"):                                 "something unusual",
	} {
		if got := plainError(err); got != want {
			t.Errorf("plainError(%v) = %q, want %q", err, got, want)
		}
	}
	if s := torErrorText(context.DeadlineExceeded, true); !strings.Contains(s, "another bridge type") {
		t.Errorf("torErrorText timeout: %q", s)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// A bridge line Arti would reject is caught before Tor starts, with a
// message the user can act on.
func TestTorBridgeWithoutFingerprintIsReported(t *testing.T) {
	var logs []string
	e := New(Config{DataDir: tempDir(t), OnLog: func(level, msg string) { logs = append(logs, level+" "+msg) }})
	p := &profile.Profile{Name: "Tor", Protocol: profile.Tor, Tor: &profile.TorOpts{
		Bridges: []string{"meek_lite 192.0.2.20:80 url=https://x.example front=y"},
	}}
	p.EnsureID()
	if err := e.Start([]*profile.Profile{p}, singbox.Options{Mode: singbox.ModeProxy, Selected: p.ID}); err == nil {
		t.Fatal("Start accepted a bridge line without a fingerprint")
	}
	st := e.State()
	if st.Status != StatusError || !strings.Contains(st.Error, "fingerprint") || strings.Contains(st.Error, "%!") {
		t.Fatalf("state = %+v", st)
	}
	t.Logf("shown to the user: %s", st.Error)
}

// With failover on, a failing Tor profile must stop with an error instead of
// quietly switching to a non-Tor profile.
func TestTorDoesNotFailOverToOtherProfiles(t *testing.T) {
	e := New(Config{DataDir: tempDir(t)})
	tp := &profile.Profile{Name: "Tor", Protocol: profile.Tor, Tor: &profile.TorOpts{Bridges: []string{"meek_lite 192.0.2.20:80 url=x"}}}
	ss := &profile.Profile{Name: "SS", Protocol: profile.Shadowsocks, Server: "127.0.0.1", Port: 1, Method: "aes-128-gcm", Password: "x"}
	tp.EnsureID()
	ss.EnsureID()
	e.Start([]*profile.Profile{tp, ss}, singbox.Options{Mode: singbox.ModeProxy, Selected: tp.ID, Failover: true})
	st := e.State()
	if st.Status != StatusError || st.Selected == ss.ID || st.Failover != nil {
		t.Fatalf("state = %+v", st)
	}
}
