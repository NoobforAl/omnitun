package engine

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/urltest"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/NoobforAl/omnitun/core/netcheck"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// FailoverEvent is reported in State when the engine switches profiles on its own.
type FailoverEvent struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

const failThreshold = 2

// profileOrder lists profile IDs in the user's order. Skipped profiles have
// no outbound, so probing them simply fails and they are passed over.
func profileOrder(profiles []*profile.Profile) []string {
	order := make([]string, 0, len(profiles))
	for _, p := range profiles {
		order = append(order, p.ID)
	}
	return order
}

// failoverLoop health-checks the selected profile. After failThreshold
// consecutive failures it checks whether the device is online at all: if not,
// it reports NoNetwork and waits; if so, it switches to the next profile (in
// the user's order) that passes a probe.
func (e *Engine) failoverLoop(ctx context.Context, instance *box.Box, order []string, opts singbox.Options) {
	interval := opts.FailoverInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	testURL := opts.TestURL
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	// The first check runs soon after start to catch a dead profile quickly.
	timer := time.NewTimer(min(3*time.Second, interval))
	defer timer.Stop()
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(interval)

		e.mu.Lock()
		if e.instance != instance {
			e.mu.Unlock()
			return
		}
		current := e.state.Selected
		e.mu.Unlock()
		if current == singbox.TagAuto || current == "" {
			continue // the urltest group already picks a working profile
		}

		err := e.probe(ctx, instance, current, testURL)
		if err == nil {
			if failures > 0 {
				e.updateState(instance, func(s *State) { s.NoNetwork = false })
			}
			failures = 0
			continue
		}
		failures++
		e.logf("warn", "health check %s failed (%d/%d): %v", e.nameLocked(current), failures, failThreshold, err)
		if failures < failThreshold {
			// Confirm quickly instead of waiting a full interval.
			timer.Reset(min(5*time.Second, interval))
			continue
		}

		if nerr := e.netCheck(ctx, instance); nerr != nil {
			e.logf("warn", "no internet connection, not switching: %v", nerr)
			e.updateState(instance, func(s *State) { s.NoNetwork = true })
			continue
		}
		e.updateState(instance, func(s *State) { s.NoNetwork = false })

		next := ""
		for _, tag := range rotateAfter(order, current) {
			if e.probe(ctx, instance, tag, testURL) == nil {
				next = tag
				break
			}
		}
		if next == "" {
			e.logf("error", "all profiles are unreachable")
			e.updateState(instance, func(s *State) { s.Error = "all profiles are unreachable" })
			continue
		}
		e.mu.Lock()
		if e.instance == instance {
			reason := fmt.Sprintf("%s stopped responding: %v", e.name(current), err)
			e.logf("warn", "failover: %s", reason)
			e.selectLocked(next, &FailoverEvent{From: current, To: next, Reason: err.Error(), At: time.Now()})
		}
		e.mu.Unlock()
		failures = 0
	}
}

// rotateAfter returns order starting after current, wrapping around, without current.
func rotateAfter(order []string, current string) []string {
	idx := -1
	for i, t := range order {
		if t == current {
			idx = i
			break
		}
	}
	out := make([]string, 0, len(order))
	for i := 1; i <= len(order); i++ {
		t := order[(idx+i+len(order))%len(order)]
		if t != current {
			out = append(out, t)
		}
	}
	return out
}

// updateState mutates the state if instance is still the running one.
func (e *Engine) updateState(instance *box.Box, f func(*State)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance != instance {
		return
	}
	s := e.state
	f(&s)
	e.setState(s)
}

// probe runs one URL test through the given profile's outbound.
func (e *Engine) probe(ctx context.Context, instance *box.Box, tag, testURL string) error {
	dialer, ok := lookupDialer(instance, tag)
	if !ok {
		return fmt.Errorf("profile %s is not running", tag)
	}
	// Endpoints that connect in the background (OpenVPN) refuse traffic until
	// their handshake finishes; wait for that instead of failing at once.
	readyBy := time.Now().Add(endpointReadyTimeout)
	// Tor builds circuits on the first connection, which takes a while.
	timeout := 8 * time.Second
	e.mu.Lock()
	if e.torRun != nil {
		timeout = 40 * time.Second
	}
	e.mu.Unlock()
	for {
		pctx, cancel := context.WithTimeout(ctx, timeout)
		_, err := urltest.URLTest(pctx, testURL, dialer)
		cancel()
		if err == nil || !strings.Contains(err.Error(), "not ready") || time.Now().After(readyBy) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// endpointReadyTimeout is how long a probe waits for an endpoint that is still connecting.
const endpointReadyTimeout = 15 * time.Second

// netCheck tests internet access through the tunnel's direct outbound, which
// bypasses the VPN (protected socket on Android).
func (e *Engine) netCheck(ctx context.Context, instance *box.Box) error {
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		direct, ok := instance.Outbound().Outbound(singbox.TagDirect)
		if !ok {
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		}
		return direct.DialContext(ctx, network, M.ParseSocksaddr(addr))
	}
	check := e.cfg.NetCheck
	if check == nil {
		check = netcheck.OnlineDial
	}
	return check(ctx, dial)
}

// name returns a profile's display name; e.mu must be held.
func (e *Engine) name(tag string) string {
	if n, ok := e.names[tag]; ok {
		return n
	}
	return tag
}

// nameLocked is name for callers that do not hold e.mu.
func (e *Engine) nameLocked(tag string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.name(tag)
}
