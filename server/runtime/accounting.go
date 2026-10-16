package runtime

import (
	"context"
	"net/netip"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/log"

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

// seenConn is what was already counted of one connection.
type seenConn struct {
	user     string
	up, down int64
}

// collector turns sing-box's per-connection counters into per-user totals.
type collector struct {
	tm *trafficcontrol.Manager
	tr *stats.Tracker
	// userByIP names the user behind a WireGuard tunnel address: the
	// endpoint does not report users, but every user has a fixed address.
	userByIP func(netip.Addr) string
	seen     map[uuid.UUID]seenConn
	done     map[uuid.UUID]bool // closed connections already counted
	last     time.Time
	started  time.Time
}

func newCollector(tm *trafficcontrol.Manager, tr *stats.Tracker, userByIP func(netip.Addr) string) *collector {
	now := time.Now()
	return &collector{tm: tm, tr: tr, userByIP: userByIP, seen: map[uuid.UUID]seenConn{}, done: map[uuid.UUID]bool{}, last: now, started: now}
}

// userOf is the user a connection belongs to: the authenticated user the
// inbound reported, or for WireGuard the owner of the source address.
func (c *collector) userOf(md *trafficcontrol.TrackerMetadata) string {
	if u := md.Metadata.User; u != "" {
		return u
	}
	if protocolKey(md.Metadata.Inbound) == "wireguard" && c.userByIP != nil && md.Metadata.Source.IsValid() {
		return c.userByIP(md.Metadata.Source.Addr)
	}
	return ""
}

// tick counts the traffic since the previous tick.
func (c *collector) tick() {
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
		c.seen[md.ID] = seenConn{user: user, up: up, down: down}
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
	c.tr.SetProtocolConnections(liveProto)

	closedNow := map[uuid.UUID]bool{}
	for _, md := range c.tm.ClosedConnections() {
		closedNow[md.ID] = true
		if c.done[md.ID] || md.ClosedAt.Before(c.started) {
			continue
		}
		up, down := md.Upload.Load(), md.Download.Load()
		prev := c.seen[md.ID] // zero when the connection opened and closed between ticks
		du, dd := up-prev.up, down-prev.down
		c.tr.AddProtocol(protocolKey(md.Metadata.Inbound), du, dd)
		if user := c.userOf(md); user != "" {
			c.tr.Add(user, du, dd)
			speed(user, du, dd)
		}
		delete(c.seen, md.ID)
		c.done[md.ID] = true
	}
	// Remember only closed connections that are still in sing-box's list.
	for id := range c.done {
		if !closedNow[id] {
			delete(c.done, id)
		}
	}
	// A connection that vanished without a closed record (list overflow) is
	// forgotten once it is no longer live.
	liveIDs := map[uuid.UUID]bool{}
	for _, md := range c.tm.Connections() {
		liveIDs[md.ID] = true
	}
	for id := range c.seen {
		if !liveIDs[id] && !closedNow[id] {
			delete(c.seen, id)
		}
	}
	c.tr.SetLive(live)
}

// accountingLoop runs for the server's whole life (not one sing-box run): it
// counts traffic while sing-box runs, and keeps enforcing quotas and saving
// the stats file also while it does not (a blocked user must be let back in).
func (s *Server) accountingLoop(ctx context.Context) {
	t := time.NewTicker(accountingInterval)
	defer t.Stop()
	for n := 1; ; n++ {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s.accMu.Lock()
		if s.coll != nil {
			s.coll.tick()
		}
		s.accMu.Unlock()
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
