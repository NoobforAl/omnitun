package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/core/tor"
)

// Bootstrap progress fills this slice of the connection ring (between
// "preparing" and "opening the tunnel").
const (
	torPercentFrom = 25
	torPercentTo   = 60
)

// torRun is a Tor client started for a selected Tor profile. sing-box starts
// once it is ready and routes the profile into its SOCKS5 port.
type torRun struct {
	profile  *profile.Profile
	profiles []*profile.Profile
	opts     singbox.Options
	cancel   context.CancelFunc
	proc     *tor.Process
}

func (r *torRun) stop() {
	r.cancel()
	if r.proc != nil {
		r.proc.Stop()
	}
}

// withoutTor drops Tor profiles from a list.
func withoutTor(profiles []*profile.Profile) []*profile.Profile {
	out := make([]*profile.Profile, 0, len(profiles))
	for _, p := range profiles {
		if p.Protocol != profile.Tor {
			out = append(out, p)
		}
	}
	return out
}

// torBridges lists the bridge lines a Tor profile uses.
func torBridges(p *profile.Profile) ([]string, error) {
	if p.Tor == nil {
		return nil, nil
	}
	bridges := append([]string{}, p.Tor.Bridges...)
	for _, b := range bridges {
		if !hasFingerprint(b) {
			return nil, errNoFingerprint
		}
	}
	if k := p.Tor.Builtin; k != "" {
		b := tor.Builtin(k)
		if len(b) == 0 {
			return nil, fmt.Errorf("unknown built-in bridge kind %q", k)
		}
		bridges = append(bridges, b...)
	}
	return bridges, nil
}

var fingerprintRE = regexp.MustCompile(`(?i)(^|\s)(fingerprint=)?[0-9a-f]{40}(\s|$)`)

// hasFingerprint reports whether a bridge line carries a relay fingerprint.
func hasFingerprint(line string) bool { return fingerprintRE.MatchString(line) }

// startTorLocked starts Tor in the background and reports its bootstrap as
// connection progress; the tunnel starts once Tor is ready. e.mu must be held.
func (e *Engine) startTorLocked(p *profile.Profile, profiles []*profile.Profile, opts singbox.Options) error {
	if opts.Mode != singbox.ModeTUN && opts.Mode != singbox.ModeProxy {
		return e.failLocked(StagePrepare, fmt.Errorf("unknown mode %q", opts.Mode))
	}
	bridges, err := torBridges(p)
	if err != nil {
		return e.failLocked(StagePrepare, err)
	}
	// Fail closed: a Tor profile never silently falls back to a non-Tor one
	// (that would drop the anonymity the user chose); the user picks another
	// profile themselves.
	opts.Failover = false
	// Tor and its bridge transports run as this app: in VPN mode keep the app
	// itself out of the VPN so they reach the network directly.
	if opts.Mode == singbox.ModeTUN && len(opts.IncludePackages) == 0 {
		opts.ExcludePackages = append(append([]string{}, opts.ExcludePackages...), singbox.SelfPackage)
	}

	ctx, cancel := context.WithTimeout(context.Background(), torAttempts*torAttemptTimeout+30*time.Second)
	run := &torRun{profile: p, profiles: profiles, opts: opts, cancel: cancel}
	e.torRun = run
	name := e.name(p.ID)
	e.logf("info", "starting Tor (%s)", name)
	e.setState(State{Status: StatusStarting, Selected: p.ID, Progress: &Progress{
		Stage: StageEngineStart, Percent: torPercentFrom, Profile: name, Detail: "Starting Tor", DetailCode: DetailTorStarting,
	}})

	// Each Tor profile keeps its own directory cache. Guard state is dropped on
	// every connect: on a network that blocks Tor, earlier failures mark every
	// guard "down" and Arti would refuse to try them again for a long time.
	torDir := filepath.Join(e.dataDir(), "tor", p.ID)
	os.RemoveAll(filepath.Join(torDir, "state"))
	go func() {
		var proc *tor.Process
		var err error
		for attempt := 1; attempt <= torAttempts; attempt++ {
			if attempt > 1 {
				e.torProgress(run, name, 0, Progress{Detail: fmt.Sprintf("Retrying Tor (%d/%d)", attempt, torAttempts), DetailCode: DetailTorRetry, DetailN: attempt, DetailM: torAttempts})
				os.RemoveAll(filepath.Join(torDir, "state"))
			}
			actx, acancel := context.WithTimeout(ctx, torAttemptTimeout)
			proc, err = e.startTorOnce(actx, run, torDir, bridges, name)
			acancel()
			if err == nil || ctx.Err() != nil || e.currentTor() != run {
				break
			}
			e.logf("warn", "Tor attempt %d/%d failed: %v", attempt, torAttempts, err)
		}
		if err == nil {
			e.mu.Lock()
			if e.torRun != run {
				e.mu.Unlock()
				proc.Stop()
				return
			}
			run.proc = proc
			e.mu.Unlock()
		} else if ctx.Err() != nil {
			err = ctx.Err()
		}

		e.mu.Lock()
		defer e.mu.Unlock()
		if e.torRun != run {
			return
		}
		if err != nil {
			e.torFailedLocked(run, err)
			return
		}
		opts := run.opts
		opts.TorSocks = run.proc.SocksAddr
		e.logf("info", "Tor is ready on %s", opts.TorSocks)
		if err := e.startSingboxLocked(run.profiles, opts, run.profile.ID); err != nil {
			run.stop()
			e.torRun = nil
			return
		}
		go e.watchTor(run)
	}()
	return nil
}

// torAttempts is how many times Tor is started before giving up: snowflake
// and bridges are intermittent, and a fresh start often succeeds at once.
const (
	torAttempts       = 3
	torAttemptTimeout = 90 * time.Second
)

func (e *Engine) currentTor() *torRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.torRun
}

// torProgress publishes Tor's bootstrap in the connection ring.
func (e *Engine) torProgress(run *torRun, name string, percent int, detail Progress) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.torRun != run || e.state.Status != StatusStarting {
		return
	}
	s := e.state
	detail.Stage, detail.Profile = StageEngineStart, name
	detail.Percent = torPercentFrom + (torPercentTo-torPercentFrom)*percent/100
	s.Progress = &detail
	e.setState(s)
}

// startTorOnce starts Tor and waits until it can carry a connection: its
// bootstrap finishes and a test connection through a fresh circuit works.
func (e *Engine) startTorOnce(ctx context.Context, run *torRun, dir string, bridges []string, name string) (*tor.Process, error) {
	// Snowflake and the other transports log through the standard logger; only their problems reach the app log.
	tor.SetTransportLog(func(line string) { e.logf("warn", "%s", line) })
	proc, err := tor.Start(tor.Config{
		DataDir: dir,
		Bridges: bridges,
		OnBootstrap: func(percent int, status string) {
			e.torProgress(run, name, percent*9/10, Progress{Detail: fmt.Sprintf("Connecting to Tor %d%%", percent), DetailCode: DetailTorBootstrap, DetailN: percent})
		},
		Logf: func(format string, args ...any) { e.logf("warn", format, args...) },
	})
	if err != nil {
		return nil, err
	}
	if err = proc.Ready(ctx); err == nil {
		e.torProgress(run, name, 92, Progress{Detail: "Building a Tor circuit", DetailCode: DetailTorCircuit})
		err = tor.CheckCircuit(ctx, proc.SocksAddr)
	}
	if err != nil {
		proc.Stop()
		return nil, err
	}
	return proc, nil
}

// watchTor ends the tunnel when the Tor process dies while connected.
func (e *Engine) watchTor(run *torRun) {
	<-run.proc.Done()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.torRun != run {
		return
	}
	e.torFailedLocked(run, errors.New("the Tor client stopped"))
}

// torFailedLocked reports a failed Tor bootstrap or a dead Tor client,
// failing over to the next profile when enabled. e.mu must be held.
func (e *Engine) torFailedLocked(run *torRun, cause error) {
	if e.torRun == run {
		e.torRun = nil
	}
	if e.instance != nil {
		e.teardownLocked()
	}
	run.stop()
	bridges, _ := torBridges(run.profile)
	reason := torErrorText(cause, len(bridges) > 0)
	e.logf("error", "%s failed: %v", stageLabel[StageEngineStart], cause)
	msg := fmt.Sprintf("%s failed: %s", stageLabel[StageEngineStart], reason)
	e.setState(State{Status: StatusError, Error: msg, Selected: run.profile.ID, Progress: progress(StageEngineStart, e.name(run.profile.ID))})
}
