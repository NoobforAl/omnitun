package runtime

import (
	"sync"
	"time"

	"github.com/NoobforAl/omnitun/server/hostproto"
	"github.com/NoobforAl/omnitun/server/stats"
)

// hostMeters turns the cumulative per-user counts of the host-level servers
// (SSTP, IKEv2, AmneziaWG) into the tracker's per-user and per-protocol
// totals, and into live connection counts and speed. sing-box's own
// connections are counted by the collector instead.
type hostMeters struct {
	mu   sync.Mutex
	by   map[string]*meterState // by protocol key
	last time.Time              // the previous reading, for the speed
	now  func() time.Time       // tests replace it
}

type meterState struct {
	meter hostproto.Meter
	last  map[string][2]uint64 // what was counted so far, by user: up, down
	live  map[string]stats.Live
}

func (h *hostMeters) clock() time.Time {
	if h.now != nil {
		return h.now()
	}
	return time.Now()
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
	now := h.clock()
	dt := now.Sub(h.last).Seconds()
	h.last = now
	if dt < 0.5 {
		dt = 0.5
	}
	for key, st := range h.by {
		h.collectLocked(key, st, tr, dt)
	}
}

// remove takes a last reading of a server that is about to stop, and forgets it.
func (h *hostMeters) remove(key string, tr *stats.Tracker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.by[key]; st != nil {
		h.collectLocked(key, st, tr, 1)
		delete(h.by, key)
	}
}

// live is what the host servers' users are doing now, as of the last reading:
// per user, and per protocol the number of connections.
func (h *hostMeters) live() (users map[string]stats.Live, protocols map[string]int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	users, protocols = map[string]stats.Live{}, map[string]int{}
	for key, st := range h.by {
		for user, l := range st.live {
			u := users[user]
			u.Connections += l.Connections
			u.UpBps += l.UpBps
			u.DownBps += l.DownBps
			users[user] = u
			protocols[key] += l.Connections
		}
	}
	return users, protocols
}

func (h *hostMeters) collectLocked(key string, st *meterState, tr *stats.Tracker, dt float64) {
	live := map[string]stats.Live{}
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
		if u.Sessions > 0 {
			live[u.User] = stats.Live{Connections: u.Sessions, UpBps: int64(float64(du) / dt), DownBps: int64(float64(dd) / dt)}
		}
		if tr == nil || (du == 0 && dd == 0) {
			continue
		}
		tr.Add(u.User, int64(du), int64(dd))
		tr.AddProtocol(key, int64(du), int64(dd))
	}
	st.live = live
}
