// Package engine runs a tunnel built from profiles and reports its state.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"

	"github.com/NoobforAl/omnitun/core/netcheck"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// Status of the engine.
type Status string

const (
	StatusStopped  Status = "disconnected"
	StatusStarting Status = "connecting"
	StatusRunning  Status = "connected"
	StatusError    Status = "error"
)

// State is a snapshot reported to the UI.
type State struct {
	Status  Status    `json:"status"`
	Since   time.Time `json:"since,omitzero"`
	Error   string    `json:"error,omitempty"`
	Skipped []string  `json:"skipped,omitempty"`
	// Selected is the active profile ID, or "auto".
	Selected  string `json:"selected,omitempty"`
	UpBps     int64  `json:"up_bps"`
	DownBps   int64  `json:"down_bps"`
	UpTotal   int64  `json:"up_total"`
	DownTotal int64  `json:"down_total"`
	// NoNetwork is set while connected but the device has no internet at all.
	NoNetwork bool `json:"no_network,omitempty"`
	// Failover describes the most recent automatic switch, if any.
	Failover *FailoverEvent `json:"failover,omitempty"`
	// Progress is the current (or, on error, the failed) connection stage.
	Progress *Progress `json:"progress,omitempty"`
}

// Config sets up an engine.
type Config struct {
	// DataDir holds the sing-box cache file. Defaults to the OS temp dir.
	DataDir string
	// OnState is called on every state change and once per second with traffic while running.
	OnState func(State)
	// OnLog receives sing-box log lines.
	OnLog func(level, message string)
	// Platform provides TUN, socket protection and interface monitoring on
	// mobile OSes (see core/mobile). Nil on desktop.
	Platform adapter.PlatformInterface
	// CertificateStore overrides sing-box's CA store ("system", "mozilla", "chrome").
	// Android needs "mozilla": Go cannot read the system store there.
	CertificateStore string
	// NetCheck reports whether the device is online, dialing through dial
	// (the tunnel's direct outbound). Nil uses netcheck.OnlineDial.
	NetCheck func(ctx context.Context, dial netcheck.DialFunc) error
	// SkipHandshake reports "connected" as soon as the tunnel is up, without
	// first reaching a server through it (tests with unreachable servers).
	SkipHandshake bool
}

// Engine owns at most one running sing-box instance.
type Engine struct {
	cfg Config

	mu       sync.Mutex
	instance *box.Box
	cancel   context.CancelFunc
	state    State
	// names maps profile IDs to display names for log messages.
	names map[string]string
}

// New returns an idle engine.
func New(cfg Config) *Engine {
	return &Engine{cfg: cfg, state: State{Status: StatusStopped}}
}

// State returns the current state.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// setState must be called with e.mu held.
func (e *Engine) setState(s State) {
	e.state = s
	if e.cfg.OnState != nil {
		e.cfg.OnState(s)
	}
}

func (e *Engine) logf(level, format string, args ...any) {
	if e.cfg.OnLog != nil {
		e.cfg.OnLog(level, fmt.Sprintf(format, args...))
	}
}

// CheckConfig validates a generated config against sing-box's schema without starting it.
func CheckConfig(cfg map[string]any) error {
	_, err := parseOptions(include.Context(context.Background()), cfg)
	return err
}

func parseOptions(ctx context.Context, cfg map[string]any) (option.Options, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return option.Options{}, err
	}
	return sjson.UnmarshalExtendedContext[option.Options](ctx, raw)
}

type logWriter struct{ e *Engine }

// ansiEscape matches terminal color codes, which sing-box adds to log lines.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (w logWriter) WriteMessage(level log.Level, message string) {
	// sing-box hands the platform writer every level, including per-connection
	// info/debug lines; forward only warnings and errors so the UI log stays readable.
	if level > log.LevelWarn {
		return
	}
	if w.e.cfg.OnLog != nil {
		w.e.cfg.OnLog(log.FormatLevel(level), ansiEscape.ReplaceAllString(message, ""))
	}
}

func (e *Engine) dataDir() string {
	if e.cfg.DataDir != "" {
		return e.cfg.DataDir
	}
	return filepath.Join(osTempDir(), "omnitun")
}

// newBox builds and creates (but does not start) a sing-box instance.
func (e *Engine) newBox(cfg map[string]any) (*box.Box, context.Context, context.CancelFunc, error) {
	if e.cfg.CertificateStore != "" {
		cfg["certificate"] = map[string]any{"store": e.cfg.CertificateStore}
	}
	cfg["experimental"] = map[string]any{
		"cache_file": map[string]any{"enabled": true, "path": filepath.Join(e.dataDir(), "cache.db")},
	}
	if err := ensureDir(e.dataDir()); err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	if e.cfg.Platform != nil {
		service.MustRegister[adapter.PlatformInterface](ctx, e.cfg.Platform)
	}
	options, err := parseOptions(ctx, cfg)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options, PlatformLogWriter: logWriter{e}})
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return instance, ctx, cancel, nil
}

// Start builds a config for profiles and starts the tunnel. A running tunnel is stopped first.
func (e *Engine) Start(profiles []*profile.Profile, opts singbox.Options) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopLocked()

	selected := opts.Selected
	if selected == "" {
		selected = singbox.TagAuto
	}
	e.names = map[string]string{singbox.TagAuto: "auto"}
	for _, p := range profiles {
		p.EnsureID()
		e.names[p.ID] = p.DisplayName()
	}
	fail := func(stage string, err error) error {
		err = fmt.Errorf("%s failed: %w", stageLabel[stage], err)
		e.logf("error", "%v", err)
		e.setState(State{Status: StatusError, Error: err.Error(), Progress: progress(stage, "")})
		return err
	}

	e.setState(State{Status: StatusStarting, Selected: selected, Progress: progress(StagePrepare, "")})
	cfg, skipped, err := singbox.Build(profiles, opts)
	if err != nil {
		return fail(StagePrepare, err)
	}
	var skippedMsgs []string
	for _, s := range skipped {
		skippedMsgs = append(skippedMsgs, s.Error())
		e.logf("warn", "skipped profile: %v", s)
	}
	e.logf("info", "starting %d profile(s) in %s mode", len(profiles)-len(skipped), orMode(opts.Mode))

	e.setState(State{Status: StatusStarting, Skipped: skippedMsgs, Selected: selected, Progress: progress(StageEngineStart, "")})
	instance, ctx, cancel, err := e.newBox(cfg)
	if err != nil {
		return fail(StageEngineStart, err)
	}
	// instance.Start opens the TUN / listening port, so failures here belong to the tunnel stage.
	if err = instance.Start(); err != nil {
		instance.Close()
		cancel()
		return fail(StageTunnelUp, err)
	}
	e.instance, e.cancel = instance, cancel
	go e.trafficLoop(ctx, instance)

	if e.cfg.SkipHandshake {
		e.setState(State{Status: StatusRunning, Since: time.Now(), Skipped: skippedMsgs, Selected: selected, Progress: progress(StageConnected, "")})
		if opts.Failover {
			go e.failoverLoop(ctx, instance, profileOrder(profiles), opts)
		}
		return nil
	}
	e.setState(State{Status: StatusStarting, Skipped: skippedMsgs, Selected: selected, Progress: progress(StageTunnelUp, "")})
	go e.handshake(ctx, instance, profileOrder(profiles), selected, opts)
	return nil
}

func orMode(m singbox.Mode) singbox.Mode {
	if m == "" {
		return singbox.ModeProxy
	}
	return m
}

// trafficLoop publishes per-second traffic rates until ctx is cancelled.
func (e *Engine) trafficLoop(ctx context.Context, instance *box.Box) {
	manager := service.PtrFromContext[trafficcontrol.Manager](ctx)
	if manager == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastUp, lastDown int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		up, down := manager.Total()
		e.mu.Lock()
		if e.instance != instance {
			e.mu.Unlock()
			return
		}
		s := e.state
		s.UpBps, s.DownBps = up-lastUp, down-lastDown
		s.UpTotal, s.DownTotal = up, down
		e.setState(s)
		e.mu.Unlock()
		lastUp, lastDown = up, down
	}
}

// Select switches the running tunnel to a profile ID (or "auto") without reconnecting.
func (e *Engine) Select(tag string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.selectLocked(tag, nil)
}

// selectLocked switches the selector; failover, when set, is recorded in the state.
func (e *Engine) selectLocked(tag string, failover *FailoverEvent) error {
	if e.instance == nil {
		return errors.New("not running")
	}
	out, ok := e.instance.Outbound().Outbound(singbox.TagSelect)
	if !ok {
		return errors.New("selector not found")
	}
	sel, ok := out.(interface{ SelectOutbound(string) bool })
	if !ok || !sel.SelectOutbound(tag) {
		return fmt.Errorf("unknown profile %q", tag)
	}
	s := e.state
	s.Selected = tag
	s.Error, s.NoNetwork = "", false
	if failover != nil {
		s.Failover = failover
	}
	e.setState(s)
	e.logf("info", "switched to %s", e.name(tag))
	return nil
}

// Stop tears down the tunnel. It is safe to call when nothing is running.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stopLocked()
}

func (e *Engine) stopLocked() error {
	if e.instance == nil {
		return nil
	}
	err := e.instance.Close()
	e.cancel()
	e.instance, e.cancel = nil, nil
	e.setState(State{Status: StatusStopped})
	e.logf("info", "stopped")
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// LatencyResult is the outcome of testing one profile.
type LatencyResult struct {
	ID    string `json:"id"`
	Ms    int    `json:"ms"` // -1 when unreachable
	Error string `json:"error,omitempty"`
}

// TestLatency measures an HTTP round trip through each profile with a
// temporary sing-box instance that has no inbounds, so it works whether or
// not a tunnel is running.
func (e *Engine) TestLatency(ctx context.Context, profiles []*profile.Profile, testURL string) ([]LatencyResult, error) {
	cfg, skipped, err := singbox.Build(profiles, singbox.Options{TestURL: testURL, LogLevel: "error"})
	if err != nil {
		return nil, err
	}
	delete(cfg, "inbounds")
	delete(cfg, "dns")
	delete(cfg, "route")
	tmp := New(Config{
		DataDir:          filepath.Join(e.dataDir(), "latency"),
		Platform:         e.cfg.Platform,
		CertificateStore: e.cfg.CertificateStore,
	})
	instance, boxCtx, cancel, err := tmp.newBox(cfg)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if err := instance.Start(); err != nil {
		instance.Close()
		return nil, err
	}
	defer instance.Close()

	results := make([]LatencyResult, len(profiles))
	var wg sync.WaitGroup
	for i, p := range profiles {
		results[i] = LatencyResult{ID: p.ID, Ms: -1}
		dialer, ok := lookupDialer(instance, p.ID)
		if !ok {
			results[i].Error = "not supported by engine"
			continue
		}
		wg.Add(1)
		go func(r *LatencyResult) {
			defer wg.Done()
			// Derive from the box context: urltest reads the configured root CA pool from it.
			tctx, tcancel := context.WithTimeout(boxCtx, 8*time.Second)
			stop := context.AfterFunc(ctx, tcancel)
			defer stop()
			defer tcancel()
			ms, err := urltest.URLTest(tctx, testURL, dialer)
			if err != nil {
				r.Error = err.Error()
				return
			}
			r.Ms = int(ms)
		}(&results[i])
	}
	wg.Wait()
	for _, r := range results {
		if r.Error != "" {
			e.logf("warn", "latency test %s: %s", r.ID, r.Error)
		}
	}
	for _, s := range skipped {
		e.logf("warn", "latency test skipped: %v", s)
	}
	return results, nil
}

func lookupDialer(instance *box.Box, tag string) (adapter.Outbound, bool) {
	if out, ok := instance.Outbound().Outbound(tag); ok {
		return out, true
	}
	if ep, ok := instance.Endpoint().Get(tag); ok {
		return ep, true
	}
	return nil, false
}
