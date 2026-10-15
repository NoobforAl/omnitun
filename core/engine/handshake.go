package engine

import (
	"context"
	"fmt"
	"time"

	box "github.com/sagernet/sing-box"

	"github.com/NoobforAl/omnitun/core/singbox"
)

// handshake reaches a server through the tunnel before reporting
// "connected". The selected profile is tried first; with failover on, the
// remaining profiles follow in list order. If none responds, the tunnel is
// torn down and the error names the stage and whether the device is offline.
func (e *Engine) handshake(ctx context.Context, instance *box.Box, order []string, selected string, opts singbox.Options) {
	testURL := opts.TestURL
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	candidates := []string{selected}
	if opts.Failover && selected != singbox.TagAuto {
		candidates = append(candidates, rotateAfter(order, selected)...)
	}

	var firstErr error
	for i, tag := range candidates {
		if !e.updateProgress(instance, progress(StageHandshake, e.nameLocked(tag))) {
			return // stopped meanwhile
		}
		// The auto group probes through the selector, which follows the urltest group.
		probeTag := tag
		if tag == singbox.TagAuto {
			probeTag = singbox.TagSelect
		}
		err := e.probe(ctx, instance, probeTag, testURL)
		if err == nil {
			e.mu.Lock()
			if e.instance != instance {
				e.mu.Unlock()
				return
			}
			if i > 0 {
				e.logf("warn", "failover: %s did not respond, using %s", e.name(selected), e.name(tag))
				e.selectLocked(tag, &FailoverEvent{From: selected, To: tag, Reason: firstErr.Error(), At: time.Now()})
			}
			s := e.state
			s.Status, s.Since, s.Error, s.Progress = StatusRunning, time.Now(), "", progress(StageConnected, e.name(tag))
			e.setState(s)
			e.mu.Unlock()
			e.logf("info", "connected via %s", e.nameLocked(tag))
			if opts.Failover {
				go e.failoverLoop(ctx, instance, order, opts)
			}
			return
		}
		if firstErr == nil {
			firstErr = err
		}
		e.logf("warn", "handshake with %s failed: %v", e.nameLocked(tag), err)
		if ctx.Err() != nil {
			return
		}
	}

	// Nothing answered: tell "offline" apart from "servers unreachable".
	msg := fmt.Sprintf("%s failed: %v", stageLabel[StageHandshake], firstErr)
	if nerr := e.netCheck(ctx, instance); nerr != nil {
		msg = "No internet connection"
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance != instance {
		return
	}
	e.logf("error", "%s", msg)
	failed := progress(StageHandshake, e.name(selected))
	e.stopLocked()
	e.setState(State{Status: StatusError, Error: msg, Progress: failed, NoNetwork: msg == "No internet connection"})
}

// updateProgress sets the progress if instance is still running; it reports whether it was.
func (e *Engine) updateProgress(instance *box.Box, p *Progress) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance != instance {
		return false
	}
	s := e.state
	s.Progress = p
	e.setState(s)
	return true
}
