// Package engine runs a tunnel built from profiles and reports its state.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"

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
}

// Engine owns at most one running sing-box instance.
type Engine struct {
	mu       sync.Mutex
	instance *box.Box
	cancel   context.CancelFunc
	state    State
	onState  func(State)
}

// New returns an engine. onState, if non-nil, is called on every state change.
func New(onState func(State)) *Engine {
	return &Engine{state: State{Status: StatusStopped}, onState: onState}
}

// State returns the current state.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *Engine) setState(s State) {
	e.state = s
	if e.onState != nil {
		e.onState(s)
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

// Start builds a config for profiles and starts the tunnel. A running tunnel is stopped first.
func (e *Engine) Start(profiles []*profile.Profile, opts singbox.Options) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopLocked()

	cfg, skipped, err := singbox.Build(profiles, opts)
	if err != nil {
		e.setState(State{Status: StatusError, Error: err.Error()})
		return err
	}
	var skippedMsgs []string
	for _, s := range skipped {
		skippedMsgs = append(skippedMsgs, s.Error())
	}
	e.setState(State{Status: StatusStarting, Skipped: skippedMsgs})

	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	options, err := parseOptions(ctx, cfg)
	if err == nil {
		var instance *box.Box
		instance, err = box.New(box.Options{Context: ctx, Options: options})
		if err == nil {
			if err = instance.Start(); err == nil {
				e.instance, e.cancel = instance, cancel
				e.setState(State{Status: StatusRunning, Since: time.Now(), Skipped: skippedMsgs})
				return nil
			}
			instance.Close()
		}
	}
	cancel()
	err = fmt.Errorf("start engine: %w", err)
	e.setState(State{Status: StatusError, Error: err.Error(), Skipped: skippedMsgs})
	return err
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
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
