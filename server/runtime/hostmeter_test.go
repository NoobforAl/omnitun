package runtime

import (
	"testing"

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
