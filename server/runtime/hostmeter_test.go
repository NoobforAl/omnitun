package runtime

import (
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/hostproto"
	"github.com/NoobforAl/omnitun/server/stats"
)

type fakeMeter struct{ usage []hostproto.Usage }

func (f *fakeMeter) Usage() []hostproto.Usage { return f.usage }

func TestHostMetersCountChangesPerUserAndProtocol(t *testing.T) {
	dir := tempDir(t)
	tr := stats.NewTracker(dir)
	defer tr.Stop()
	var h hostMeters
	m := &fakeMeter{}
	h.add("sstp", m)

	m.usage = []hostproto.Usage{{User: "alice", Up: 100, Down: 400}, {User: "bob", Up: 10, Down: 20}}
	h.collect(tr)
	if got := tr.MonthTotal("alice"); got != 500 {
		t.Fatalf("alice after the first reading: %d", got)
	}
	// Only the change since the last reading is added.
	m.usage = []hostproto.Usage{{User: "alice", Up: 150, Down: 450}, {User: "bob", Up: 10, Down: 20}}
	h.collect(tr)
	h.collect(tr) // a repeated reading adds nothing
	if got := tr.MonthTotal("alice"); got != 600 {
		t.Fatalf("alice after the second reading: %d", got)
	}
	if got := tr.MonthTotal("bob"); got != 30 {
		t.Fatalf("bob: %d", got)
	}

	// The last reading before a server stops is taken too.
	m.usage = []hostproto.Usage{{User: "alice", Up: 160, Down: 450}}
	h.remove("sstp", tr)
	if got := tr.MonthTotal("alice"); got != 610 {
		t.Fatalf("alice after the final reading: %d", got)
	}
	h.collect(tr) // nothing is read from a removed server
	if got := tr.MonthTotal("alice"); got != 610 {
		t.Fatalf("alice after removal: %d", got)
	}

	// A restarted server counts from zero again.
	m.usage = []hostproto.Usage{{User: "alice", Up: 5, Down: 5}}
	h.add("sstp", m)
	h.collect(tr)
	if got := tr.MonthTotal("alice"); got != 620 {
		t.Fatalf("alice after a restart: %d", got)
	}
}

func TestHostMetersLiveConnectionsAndSpeed(t *testing.T) {
	tr := stats.NewTracker(tempDir(t))
	defer tr.Stop()
	clock := time.Unix(1000, 0)
	h := hostMeters{now: func() time.Time { return clock }}
	sstp, awg := &fakeMeter{}, &fakeMeter{}
	h.add("sstp", sstp)
	h.add("amneziawg", awg)

	sstp.usage = []hostproto.Usage{{User: "alice", Up: 0, Down: 0, Sessions: 2}}
	awg.usage = []hostproto.Usage{{User: "alice", Sessions: 1}, {User: "bob", Sessions: 0}}
	h.collect(tr)

	clock = clock.Add(2 * time.Second)
	sstp.usage = []hostproto.Usage{{User: "alice", Up: 2000, Down: 6000, Sessions: 2}}
	awg.usage = []hostproto.Usage{{User: "alice", Up: 400, Down: 200, Sessions: 1}, {User: "bob"}}
	h.collect(tr)

	users, protocols := h.live()
	got := users["alice"]
	if got.Connections != 3 || got.UpBps != 1200 || got.DownBps != 3100 {
		t.Fatalf("alice live: %+v", got)
	}
	if _, ok := users["bob"]; ok {
		t.Fatalf("bob has no connection but is listed: %+v", users["bob"])
	}
	if protocols["sstp"] != 2 || protocols["amneziawg"] != 1 {
		t.Fatalf("protocols: %v", protocols)
	}

	// Everyone leaves: nothing is live any more, though the totals stay.
	clock = clock.Add(2 * time.Second)
	sstp.usage = []hostproto.Usage{{User: "alice", Up: 2000, Down: 6000}}
	awg.usage = []hostproto.Usage{{User: "alice", Up: 400, Down: 200}}
	h.collect(tr)
	if users, protocols := h.live(); len(users) != 0 || len(protocols) != 0 {
		t.Fatalf("still live: %v %v", users, protocols)
	}
	if got := tr.MonthTotal("alice"); got != 8600 {
		t.Fatalf("alice total %d", got)
	}
}
