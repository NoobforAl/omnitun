package runtime

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/observable"

	"github.com/NoobforAl/omnitun/server/logs"
	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

// accountingInterval is the counting period; enforceEvery is how many ticks
// pass between quota checks and saves of the stats file (variables so tests
// can speed them up).
var (
	accountingInterval = 2 * time.Second
	enforceEvery       = 5
)

// sbLog is the platform log writer sing-box needs before it creates its
// traffic manager; it also puts sing-box's own messages (warnings and errors,
// at the configured level) into server.log. The normal log output continues.
type sbLog struct{ file *logs.Log }

func (w sbLog) WriteMessage(level log.Level, message string) {
	w.file.Write("sing-box "+log.FormatLevel(level), message)
}

// protocolKey maps an inbound tag to the protocol it serves: ShadowTLS hands
// its streams to an internal Shadowsocks inbound.
func protocolKey(inbound string) string {
	if inbound == "shadowtls-ss" {
		return "shadowtls"
	}
	return inbound
}

// tunnelProtocol is true for the protocols whose users are told apart by the
// tunnel address their connections come from.
func tunnelProtocol(key string) bool {
	return key == "wireguard" || key == "openvpn-udp" || key == "openvpn-tcp"
}

// seenConn is what was already counted of one connection.
type seenConn struct {
	user     string
	up, down int64
	at       time.Time // last tick that saw it live
}

// countedFor is how long a closed connection is remembered as counted, so
// the closed-list pass and the close event never count it twice.
const countedFor = 5 * time.Minute

// collector turns sing-box's per-connection counters into per-user totals.
//
// Connections are polled while they live, and counted for good when they
// close. sing-box's list of closed connections is not enough for that: it is
// emptied at every garbage collection, so a connection that closed shortly
// before one would be lost. The close events (delivered as they happen) are
// the main source; the list is the fallback for events that were dropped.
type collector struct {
	tm *trafficcontrol.Manager
	tr *stats.Tracker
	// userByIP names the user behind a WireGuard or OpenVPN tunnel address:
	// those endpoints do not report users with their connections.
	userByIP func(netip.Addr) string
	last     time.Time
	started  time.Time

	mu   sync.Mutex // guards seen and done; held by tick and by the event reader
	seen map[uuid.UUID]seenConn
	done map[uuid.UUID]time.Time // closed connections already counted, and when

	events  observable.Subscription[trafficcontrol.ConnectionEvent]
	stopped chan struct{}
	drained chan struct{}
}

func newCollector(tm *trafficcontrol.Manager, tr *stats.Tracker, userByIP func(netip.Addr) string) *collector {
	now := time.Now()
	c := &collector{
		tm: tm, tr: tr, userByIP: userByIP, last: now, started: now,
		seen: map[uuid.UUID]seenConn{}, done: map[uuid.UUID]time.Time{},
		stopped: make(chan struct{}), drained: make(chan struct{}),
	}
	if events, _, err := tm.SubscribeEvents(); err == nil {
		c.events = events
		go c.readEvents()
	} else {
		close(c.drained)
	}
	return c
}

// readEvents counts every connection as it closes.
func (c *collector) readEvents() {
	defer close(c.drained)
	for {
		select {
		case ev := <-c.events:
			if ev.Type == trafficcontrol.ConnectionEventClosed && ev.Metadata != nil {
				c.mu.Lock()
				c.countClosedLocked(ev.Metadata)
				c.mu.Unlock()
			}
		case <-c.stopped:
			// Take what is already queued (the shutdown closes everything), then end.
			for {
				select {
				case ev := <-c.events:
					if ev.Type == trafficcontrol.ConnectionEventClosed && ev.Metadata != nil {
						c.mu.Lock()
						c.countClosedLocked(ev.Metadata)
						c.mu.Unlock()
					}
				case <-time.After(100 * time.Millisecond):
					return
				}
			}
		}
	}
}

// finish ends the event reader after sing-box has closed its connections.
func (c *collector) finish() {
	close(c.stopped)
	<-c.drained
	c.tm.UnSubscribeEvents(c.events)
}

// countClosedLocked counts what a closed connection moved since it was last
// seen, once. It returns the user and the bytes (for the live speed).
func (c *collector) countClosedLocked(md *trafficcontrol.TrackerMetadata) (user string, du, dd int64, counted bool) {
	if _, ok := c.done[md.ID]; ok || md.ClosedAt.Before(c.started) {
		return "", 0, 0, false
	}
	up, down := md.Upload.Load(), md.Download.Load()
	prev := c.seen[md.ID] // zero when the connection opened and closed between ticks
	du, dd = up-prev.up, down-prev.down
	c.tr.AddProtocol(protocolKey(md.Metadata.Inbound), du, dd)
	user = c.userOf(md)
	if user != "" {
		c.tr.Add(user, du, dd)
	}
	delete(c.seen, md.ID)
	c.done[md.ID] = time.Now()
	return user, du, dd, true
}

// userOf is the user a connection belongs to: the authenticated user the
// inbound reported, or for WireGuard and OpenVPN the owner of the source address.
func (c *collector) userOf(md *trafficcontrol.TrackerMetadata) string {
	if u := md.Metadata.User; u != "" {
		return u
	}
	if tunnelProtocol(protocolKey(md.Metadata.Inbound)) && c.userByIP != nil && md.Metadata.Source.IsValid() {
		return c.userByIP(md.Metadata.Source.Addr)
	}
	return ""
}

// tick counts the traffic since the previous tick. It returns what sing-box's
// users are doing now (connections and speed per user, connections per
// protocol); the caller publishes it together with the host servers' numbers.
func (c *collector) tick() (map[string]stats.Live, map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	dt := now.Sub(c.last).Seconds()
	if dt < 0.5 {
		dt = 0.5
	}
	c.last = now
	live := map[string]stats.Live{}
	speed := func(user string, up, down int64) {
		l := live[user]
		l.UpBps += int64(float64(up) / dt)
		l.DownBps += int64(float64(down) / dt)
		live[user] = l
	}

	liveProto := map[string]int{}
	for _, md := range c.tm.Connections() {
		up, down := md.Upload.Load(), md.Download.Load()
		prev := c.seen[md.ID]
		du, dd := up-prev.up, down-prev.down
		user, proto := c.userOf(md), protocolKey(md.Metadata.Inbound)
		c.seen[md.ID] = seenConn{user: user, up: up, down: down, at: now}
		c.tr.AddProtocol(proto, du, dd)
		if proto != "" {
			liveProto[proto]++
		}
		if user == "" {
			continue
		}
		c.tr.Add(user, du, dd)
		speed(user, du, dd)
		l := live[user]
		l.Connections++
		live[user] = l
	}

	// The fallback: closed connections still in sing-box's list that no event
	// counted (an event can be dropped when a burst fills its queue).
	closedNow := map[uuid.UUID]bool{}
	for _, md := range c.tm.ClosedConnections() {
		closedNow[md.ID] = true
		if user, du, dd, counted := c.countClosedLocked(md); counted && user != "" {
			speed(user, du, dd)
		}
	}
	for id, at := range c.done {
		if now.Sub(at) > countedFor {
			delete(c.done, id)
		}
	}
	// A connection that vanished without a close event or record (both
	// dropped) is forgotten after a while; the delay leaves time for a queued
	// event to arrive, which needs what was seen of the connection.
	for id, sc := range c.seen {
		if !closedNow[id] && now.Sub(sc.at) > time.Minute {
			delete(c.seen, id)
		}
	}
	return live, liveProto
}

// publishLive tells the tracker what everyone is doing now: sing-box's
// connections, as the collector counted them, and those of the host servers.
func (s *Server) publishLive(box map[string]stats.Live, boxProto map[string]int) {
	hostLive, hostProto := s.meters.live()
	users := map[string]stats.Live{}
	for _, m := range []map[string]stats.Live{box, hostLive} {
		for name, l := range m {
			u := users[name]
			u.Connections += l.Connections
			u.UpBps += l.UpBps
			u.DownBps += l.DownBps
			users[name] = u
		}
	}
	protocols := map[string]int{}
	for _, m := range []map[string]int{boxProto, hostProto} {
		for key, n := range m {
			protocols[key] += n
		}
	}
	s.tracker.SetLive(users)
	s.tracker.SetProtocolConnections(protocols)
}

// accountingLoop runs for the server's whole life (not one sing-box run): it
// counts traffic while sing-box runs, and keeps enforcing quotas and saving
// the stats file also while it does not (a blocked user must be let back in).
func (s *Server) accountingLoop(ctx context.Context) {
	t := time.NewTicker(accountingInterval)
	defer t.Stop()
	// Write the file at once, so `status` sees a running server straight away.
	if err := s.tracker.Save(); err != nil {
		s.logf("save stats: %v", err)
	}
	for n := 1; ; n++ {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		var boxLive map[string]stats.Live
		var boxProto map[string]int
		s.accMu.Lock()
		if s.coll != nil {
			boxLive, boxProto = s.coll.tick()
		}
		s.accMu.Unlock()
		s.meters.collect(s.tracker)
		s.publishLive(boxLive, boxProto)
		if n%enforceEvery == 0 {
			s.enforceQuotas()
			if err := s.tracker.Save(); err != nil {
				s.logf("save stats: %v", err)
			}
		}
	}
}

// enforceQuotas applies usage-reset requests and blocks users who used up
// their monthly quota (and lets them back in when it no longer applies). The
// change is written to the state file, which makes the server reload.
func (s *Server) enforceQuotas() {
	st, err := s.store.Load()
	if err != nil {
		return
	}
	s.tracker.Forget(func(name string) bool { return st.User(name) != nil })
	s.setWireGuardUsers(st)
	type change struct {
		name     string
		exceeded bool
	}
	var changes []change
	for _, u := range st.Users {
		if u.UsageResetAt != nil {
			s.tracker.ApplyReset(u.Name, *u.UsageResetAt)
		}
		exceeded := u.QuotaGB > 0 && float64(s.tracker.MonthTotal(u.Name)) >= u.QuotaGB*stats.GiB
		if exceeded != u.QuotaExceeded {
			changes = append(changes, change{u.Name, exceeded})
		}
	}
	if len(changes) == 0 {
		return
	}
	err = s.store.Update(func(st *store.State) error {
		for _, c := range changes {
			if u := st.User(c.name); u != nil {
				u.QuotaExceeded = c.exceeded
			}
		}
		return nil
	})
	if err != nil {
		s.logf("update quota flags: %v", err)
		return
	}
	for _, c := range changes {
		if c.exceeded {
			s.logf("user %s used up the monthly quota; blocked until it resets or is raised", c.name)
		} else {
			s.logf("user %s is under the quota again; allowed", c.name)
		}
	}
}
