package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/NoobforAl/omnitun/core/classicvpn"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// classicDialTimeout bounds the SSTP / IKEv2 handshake.
const classicDialTimeout = 25 * time.Second

// classicRun is an SSTP or IKEv2 tunnel run in the core (veepin).
type classicRun struct {
	profile  *profile.Profile
	profiles []*profile.Profile
	opts     singbox.Options
	cancel   context.CancelFunc
	tunnel   *classicvpn.Tunnel
}

func (r *classicRun) stop() {
	r.cancel()
	if r.tunnel != nil {
		r.tunnel.Close()
	}
}

// usesClassic reports whether the core runs p with classicvpn.
func usesClassic(p *profile.Profile) bool {
	if p.Protocol == profile.IKEv2 && p.IKEv2 != nil && p.IKEv2.System {
		return false
	}
	return classicvpn.Supported(p.Protocol)
}

// classicTunRequest describes the VPN interface for the negotiated addresses.
func classicTunRequest(p *profile.Profile, iface classicvpn.Interface, opts singbox.Options) TunRequest {
	req := TunRequest{
		MTU: uint32(iface.MTU), AutoRoute: true,
		Inet4Address: []string{}, Inet6Address: []string{},
		Inet4RouteRange: []string{"0.0.0.0/0"}, Inet6RouteRange: []string{},
		DNSServers:     []string{},
		IncludePackage: append([]string{}, opts.IncludePackages...),
		// The protocol's own sockets (UDP/TCP to the server) come from this
		// app; keeping the app out of the VPN keeps them off the tunnel.
		ExcludeSelf: true,
	}
	if req.MTU == 0 {
		req.MTU = 1400
	}
	if p.IKEv2 != nil && p.IKEv2.MTU >= 1280 {
		req.MTU = uint32(p.IKEv2.MTU)
	}
	if len(req.IncludePackage) == 0 {
		req.ExcludePackage = append([]string{}, opts.ExcludePackages...)
	}
	if a := classicvpn.IPv4Prefix(iface); a != "" {
		req.Inet4Address = append(req.Inet4Address, a)
	}
	if iface.IPv6 != nil && iface.IPv6.To4() == nil {
		bits := iface.Prefix6
		if bits <= 0 || bits > 128 {
			bits = 128
		}
		req.Inet6Address = append(req.Inet6Address, fmt.Sprintf("%s/%d", iface.IPv6, bits))
		req.Inet6RouteRange = append(req.Inet6RouteRange, "::/0")
	}
	for _, d := range iface.DNS {
		req.DNSServers = append(req.DNSServers, d.String())
	}
	if len(req.DNSServers) == 0 {
		req.DNSServers = []string{"1.1.1.1"}
	}
	if !opts.ProxyPrivate {
		applyLocalBypass(&req, nil)
	}
	return req
}

// startClassicLocked begins an SSTP/IKEv2 connection. The handshake runs in
// the background; e.mu must be held.
func (e *Engine) startClassicLocked(p *profile.Profile, profiles []*profile.Profile, opts singbox.Options) error {
	if opts.Mode != singbox.ModeTUN || e.cfg.Tun == nil {
		err := fmt.Errorf("%s failed: %s works in VPN mode on Android (desktop support comes later)", stageLabel[StagePrepare], p.Protocol)
		e.logf("error", "%v", err)
		e.setState(State{Status: StatusError, Error: err.Error(), Selected: p.ID, Progress: progress(StagePrepare, e.name(p.ID))})
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &classicRun{profile: p, profiles: profiles, opts: opts, cancel: cancel}
	e.classicRun = run
	e.logf("info", "starting %s (%s)", e.name(p.ID), p.Protocol)
	e.setState(State{Status: StatusStarting, Selected: p.ID, Progress: progress(StageHandshake, e.name(p.ID))})
	go e.classicDial(ctx, run)
	return nil
}

func (e *Engine) classicDial(ctx context.Context, run *classicRun) {
	dctx, cancel := context.WithTimeout(ctx, classicDialTimeout)
	defer cancel()
	open := func(iface classicvpn.Interface) (classicvpn.Device, error) {
		fd, err := e.cfg.Tun.OpenTun(classicTunRequest(run.profile, iface, run.opts))
		if err != nil {
			return nil, err
		}
		return tunFile(fd)
	}
	tunnel, err := classicvpn.Dial(dctx, run.profile, open, func(format string, args ...any) {
		e.logf("warn", "%s: "+format, append([]any{run.profile.Protocol}, args...)...)
	})

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.classicRun != run {
		if tunnel != nil {
			tunnel.Close()
		}
		return
	}
	if err != nil {
		e.classicFailedLocked(run, err)
		return
	}
	run.tunnel = tunnel
	e.logf("info", "connected via %s", e.name(run.profile.ID))
	e.setState(State{Status: StatusRunning, Since: time.Now(), Selected: run.profile.ID, Progress: progress(StageConnected, e.name(run.profile.ID))})
	go e.classicLoop(ctx, run)
}

// classicLoop publishes traffic and notices when the tunnel ends.
func (e *Engine) classicLoop(ctx context.Context, run *classicRun) {
	ended := make(chan error, 1)
	go func() { ended <- run.tunnel.Wait(ctx) }()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastRx, lastTx int64
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-ended:
			e.mu.Lock()
			if e.classicRun == run {
				if err == nil {
					err = errors.New("the server closed the connection")
				}
				e.classicFailedLocked(run, err)
			}
			e.mu.Unlock()
			return
		case <-ticker.C:
		}
		rx, tx := run.tunnel.Traffic()
		e.mu.Lock()
		if e.classicRun != run {
			e.mu.Unlock()
			return
		}
		s := e.state
		s.UpBps, s.DownBps, s.UpTotal, s.DownTotal = tx-lastTx, rx-lastRx, tx, rx
		e.setState(s)
		e.mu.Unlock()
		lastRx, lastTx = rx, tx
	}
}

// classicFailedLocked reports a failed or dropped tunnel, failing over to the
// next profile when enabled. e.mu must be held.
func (e *Engine) classicFailedLocked(run *classicRun, cause error) {
	wasRunning := e.state.Status == StatusRunning
	e.classicRun = nil
	run.stop()
	if run.opts.Failover && len(run.profiles) > 1 {
		next := rotateAfter(profileOrder(run.profiles), run.profile.ID)[0]
		e.logf("warn", "failover: %s: %v, trying %s", e.name(run.profile.ID), cause, e.name(next))
		e.carryFailover = &FailoverEvent{From: run.profile.ID, To: next, Reason: cause.Error(), At: time.Now()}
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
	e.logf("error", "%s failed: %v", stageLabel[StageHandshake], cause)
	msg := fmt.Sprintf("%s failed: %s", stageLabel[StageHandshake], plainError(cause))
	e.setState(State{Status: StatusError, Error: msg, Selected: run.profile.ID, Progress: progress(stage, e.name(run.profile.ID))})
}

// tunFile wraps a duplicate of the platform's TUN fd as a packet device.
func tunFile(fd int) (classicvpn.Device, error) {
	dup, err := dupFD(fd)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(dup), "tun")
	if f == nil {
		return nil, errors.New("invalid TUN descriptor")
	}
	return f, nil
}
