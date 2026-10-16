package runtime

import (
	"sync"

	"github.com/NoobforAl/omnitun/server/hostproto"
	"github.com/NoobforAl/omnitun/server/stats"
)

// hostMeters turns the cumulative per-user counts of the host-level servers
// (SSTP, IKEv2, AmneziaWG) into the tracker's per-user and per-protocol
// totals. sing-box's own connections are counted by the collector instead.
type hostMeters struct {
	mu sync.Mutex
	by map[string]*meterState // by protocol key
}

type meterState struct {
	meter hostproto.Meter
	last  map[string][2]uint64 // what was counted so far, by user: up, down
}

// add starts counting a server that has just been started (its counts begin at zero).
func (h *hostMeters) add(key string, m hostproto.Meter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.by == nil {
		h.by = map[string]*meterState{}
	}
	h.by[key] = &meterState{meter: m, last: map[string][2]uint64{}}
}

// collect adds what every server moved since the last reading.
func (h *hostMeters) collect(tr *stats.Tracker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for key, st := range h.by {
		h.collectLocked(key, st, tr)
	}
}

// remove takes a last reading of a server that is about to stop, and forgets it.
func (h *hostMeters) remove(key string, tr *stats.Tracker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.by[key]; st != nil {
		h.collectLocked(key, st, tr)
		delete(h.by, key)
	}
}

func (h *hostMeters) collectLocked(key string, st *meterState, tr *stats.Tracker) {
	if tr == nil {
		return
	}
	for _, u := range st.meter.Usage() {
		prev := st.last[u.User]
		du, dd := u.Up-prev[0], u.Down-prev[1]
		if u.Up < prev[0] {
			du = u.Up
		}
		if u.Down < prev[1] {
			dd = u.Down
		}
		st.last[u.User] = [2]uint64{u.Up, u.Down}
		if du == 0 && dd == 0 {
			continue
		}
		tr.Add(u.User, int64(du), int64(dd))
		tr.AddProtocol(key, int64(du), int64(dd))
	}
}
