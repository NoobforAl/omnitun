package tor

import (
	"bytes"
	"log"
	"strings"
	"sync"
)

// Snowflake (like the other pluggable transports) writes a line to the standard
// logger for every proxy it collects, every broker round trip and every ICE
// step. Left alone that is a stream of chatter on stderr (logcat on Android) at
// every connection, and it names the broker and its front domain. The core
// does not use the standard logger itself, so it is pointed at a filter that
// drops the routine lines and passes on the ones that say something went
// wrong.

var (
	ptLogMu   sync.Mutex
	ptLogSink func(line string)
)

// SetTransportLog sets where the problem lines of the pluggable transports go
// (nil drops them too). It also takes the standard logger away from stderr.
func SetTransportLog(sink func(line string)) {
	ptLogMu.Lock()
	ptLogSink = sink
	ptLogMu.Unlock()
	log.SetFlags(0)
	log.SetOutput(ptLogFilter{})
}

type ptLogFilter struct{}

// ptProblem marks a line as worth keeping.
var ptProblem = []string{"error", "fail", "timed out", "timeout", "refused", "unable", "cannot", "can't", "rejected", "no such host", "unreachable", "panic"}

func (ptLogFilter) Write(p []byte) (int, error) {
	ptLogMu.Lock()
	sink := ptLogSink
	ptLogMu.Unlock()
	if sink == nil {
		return len(p), nil
	}
	for line := range bytes.SplitSeq(p, []byte("\n")) {
		text := strings.TrimSpace(string(line))
		if text != "" && isTransportProblem(text) {
			sink("transport: " + text)
		}
	}
	return len(p), nil
}

func isTransportProblem(line string) bool {
	lower := strings.ToLower(line)
	for _, word := range ptProblem {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
