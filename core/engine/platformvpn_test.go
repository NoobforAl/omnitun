//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

type fakePlatform struct {
	mu      sync.Mutex
	started []string
	stopped int
}

func (f *fakePlatform) StartPlatformVPN(j string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, j)
	return nil
}

func (f *fakePlatform) StopPlatformVPN() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped++
}

func TestPlatformVPN(t *testing.T) {
	res, err := importer.Import("ikev2://alice:pw@vpn.example.com?remote_id=vpn.example.com&system=1#ike\n" +
		fmt.Sprintf("ss://aes-128-gcm:pw@127.0.0.1:%d#ss", freePort(t)))
	if err != nil || len(res.Profiles) != 2 {
		t.Fatalf("import: %v %+v", err, res)
	}
	ike, ss := res.Profiles[0], res.Profiles[1]
	if ike.Protocol != profile.IKEv2 || ike.IKEv2.Auth != "eap" || ike.Username != "alice" {
		t.Fatalf("ikev2 profile: %+v %+v", ike, ike.IKEv2)
	}

	fp := &fakePlatform{}
	var mu sync.Mutex
	var last State
	e := New(Config{DataDir: t.TempDir(), PlatformVPN: fp, SkipHandshake: true, OnState: func(s State) {
		mu.Lock()
		last = s
		mu.Unlock()
	}})
	state := func() State { mu.Lock(); defer mu.Unlock(); return last }

	// Connect, then the platform reports success.
	if err := e.Start(res.Profiles, singbox.Options{Selected: ike.ID, ListenPort: freePort(t)}); err != nil {
		t.Fatal(err)
	}
	if len(fp.started) != 1 || state().Status != StatusStarting {
		t.Fatalf("started=%d state=%+v", len(fp.started), state())
	}
	var sent profile.Profile
	if err := json.Unmarshal([]byte(fp.started[0]), &sent); err != nil || sent.IKEv2 == nil || sent.Server != "vpn.example.com" {
		t.Fatalf("platform got %s", fp.started[0])
	}
	e.PlatformVPNStatus("connected", "")
	if s := state(); s.Status != StatusRunning || s.ProfileName != "ike" || e.Selected() != ike.ID {
		t.Fatalf("after connected: %+v", s)
	}
	e.Stop()
	if fp.stopped != 1 || state().Status != StatusStopped {
		t.Fatalf("stop: stopped=%d %+v", fp.stopped, state())
	}

	// Failure without failover is an error.
	if err := e.Start(res.Profiles, singbox.Options{Selected: ike.ID, ListenPort: freePort(t)}); err != nil {
		t.Fatal(err)
	}
	e.PlatformVPNStatus("failed", "authentication failed")
	if s := state(); s.Status != StatusError || s.Error == "" {
		t.Fatalf("after failure: %+v", s)
	}

	// With failover, the next profile (Shadowsocks via sing-box) takes over.
	if err := e.Start(res.Profiles, singbox.Options{Selected: ike.ID, Failover: true, ListenPort: freePort(t)}); err != nil {
		t.Fatal(err)
	}
	e.PlatformVPNStatus("failed", "server unreachable")
	s := state()
	if s.Status != StatusRunning || s.Selected != ss.ID || s.Failover == nil || s.Failover.From != ike.ID {
		t.Fatalf("after failover: %+v", s)
	}
	e.Stop()
}
