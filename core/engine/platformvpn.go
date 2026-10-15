package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// PlatformVPN runs tunnels the OS implements itself (Android's IKEv2 client).
// Start must return quickly; the platform reports progress through
// Engine.PlatformVPNStatus.
type PlatformVPN interface {
	StartPlatformVPN(profileJSON string) error
	StopPlatformVPN()
}

// platformConnectTimeout is how long the OS gets to bring the tunnel up.
const platformConnectTimeout = 30 * time.Second

// isPlatformProtocol reports whether the OS, not sing-box, runs a protocol.
// IKEv2 profiles use it only when they ask for the system client (see usesClassic).
func isPlatformProtocol(proto string) bool { return proto == profile.IKEv2 }

// platformRun is a tunnel owned by the OS.
type platformRun struct {
	profile  *profile.Profile
	profiles []*profile.Profile
	opts     singbox.Options
	timer    *time.Timer
}

// startPlatformLocked asks the OS to start p. e.mu must be held.
func (e *Engine) startPlatformLocked(p *profile.Profile, profiles []*profile.Profile, opts singbox.Options) error {
	fail := func(stage string, err error) error {
		err = fmt.Errorf("%s failed: %w", stageLabel[stage], err)
		e.logf("error", "%v", err)
		e.setState(State{Status: StatusError, Error: err.Error(), Selected: p.ID, Progress: progress(stage, e.name(p.ID))})
		return err
	}
	if e.cfg.PlatformVPN == nil {
		return fail(StagePrepare, errors.New("IKEv2 needs Android 11 or newer (the system's IKEv2 client)"))
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return fail(StagePrepare, err)
	}
	e.logf("info", "starting %s (system IKEv2 client)", e.name(p.ID))
	run := &platformRun{profile: p, profiles: profiles, opts: opts}
	e.platformRun = run
	e.setState(State{Status: StatusStarting, Selected: p.ID, Progress: progress(StageHandshake, e.name(p.ID))})
	if err := e.cfg.PlatformVPN.StartPlatformVPN(string(raw)); err != nil {
		e.platformRun = nil
		return fail(StageTunnelUp, err)
	}
	run.timer = time.AfterFunc(platformConnectTimeout, func() {
		e.PlatformVPNStatus("failed", fmt.Sprintf("no connection within %s", platformConnectTimeout))
	})
	return nil
}

// PlatformVPNStatus is called by the platform when its tunnel changes:
// status is "connected", "disconnected" or "failed" (with a message).
func (e *Engine) PlatformVPNStatus(status, message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run := e.platformRun
	if run == nil {
		return
	}
	switch status {
	case "connected":
		run.timer.Stop()
		if e.state.Status != StatusRunning {
			e.logf("info", "connected via %s", e.name(run.profile.ID))
			e.setState(State{Status: StatusRunning, Since: time.Now(), Selected: run.profile.ID, Progress: progress(StageConnected, e.name(run.profile.ID))})
		}
	case "failed", "disconnected":
		run.timer.Stop()
		wasRunning := e.state.Status == StatusRunning
		reason := message
		if reason == "" {
			reason = "the system VPN stopped"
		}
		e.platformRun = nil
		e.cfg.PlatformVPN.StopPlatformVPN()
		if run.opts.Failover && len(run.profiles) > 1 {
			next := rotateAfter(profileOrder(run.profiles), run.profile.ID)[0]
			e.logf("warn", "failover: %s: %s, trying %s", e.name(run.profile.ID), reason, e.name(next))
			e.carryFailover = &FailoverEvent{From: run.profile.ID, To: next, Reason: reason, At: time.Now()}
			opts := run.opts
			opts.Selected = next
			rest := make([]*profile.Profile, 0, len(run.profiles)-1)
			for _, p := range run.profiles {
				if p.ID != run.profile.ID {
					rest = append(rest, p)
				}
			}
			e.startLocked(rest, opts)
			return
		}
		stage := StageHandshake
		if wasRunning {
			stage = StageConnected
		}
		err := fmt.Sprintf("%s failed: %s", stageLabel[StageHandshake], reason)
		e.logf("error", "%s", err)
		e.setState(State{Status: StatusError, Error: err, Selected: run.profile.ID, Progress: progress(stage, e.name(run.profile.ID))})
	}
}
