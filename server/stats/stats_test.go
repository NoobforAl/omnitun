package stats

import (
	"testing"
	"time"
)

func TestCountSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	tr := NewTracker(dir)
	tr.Add("alice", 100, 900)
	tr.Add("alice", 1, 2)
	tr.Add("", 5, 5) // unknown user: ignored
	tr.SetLive(map[string]Live{"alice": {Connections: 2, UpBps: 10, DownBps: 20}})
	if err := tr.Save(); err != nil {
		t.Fatal(err)
	}

	snap, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := snap.Users["alice"]
	if a == nil || a.Up != 101 || a.Down != 902 || a.MonthTotal() != 1003 || a.Connections != 2 || a.DownBps != 20 {
		t.Fatalf("alice = %+v", a)
	}
	if len(snap.Users) != 1 || !snap.Running(time.Now()) {
		t.Fatalf("snapshot = %+v", snap)
	}

	// A new tracker (server restart) keeps the totals.
	tr2 := NewTracker(dir)
	tr2.Add("alice", 1, 1)
	if got := tr2.MonthTotal("alice"); got != 1005 {
		t.Fatalf("after restart month total = %d", got)
	}
}

func TestMonthRollover(t *testing.T) {
	tr := NewTracker(t.TempDir())
	now := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	tr.now = func() time.Time { return now }
	tr.Add("bob", 10, 20)
	if tr.MonthTotal("bob") != 30 {
		t.Fatal("october total")
	}
	now = now.Add(2 * time.Hour) // November
	if tr.MonthTotal("bob") != 0 {
		t.Fatal("the monthly counter did not restart")
	}
	tr.Add("bob", 1, 1)
	if u := tr.snap.Users["bob"]; u.Up != 11 || u.Down != 21 || u.MonthTotal() != 2 {
		t.Fatalf("bob = %+v", u)
	}
}

func TestApplyReset(t *testing.T) {
	tr := NewTracker(t.TempDir())
	tr.Add("carol", 50, 50)
	asked := time.Now().Add(time.Minute)
	if !tr.ApplyReset("carol", asked) || tr.MonthTotal("carol") != 0 {
		t.Fatal("reset not applied")
	}
	tr.Add("carol", 5, 5)
	if tr.ApplyReset("carol", asked) || tr.MonthTotal("carol") != 10 {
		t.Fatal("the same reset request was applied twice")
	}
	if tr.ApplyReset("carol", time.Time{}) {
		t.Fatal("a zero request counted as a reset")
	}
	if u := tr.snap.Users["carol"]; u.Up != 55 {
		t.Fatalf("lifetime total changed: %+v", u)
	}
}

func TestStopAndForget(t *testing.T) {
	dir := t.TempDir()
	tr := NewTracker(dir)
	tr.Add("dave", 1, 1)
	tr.Add("gone", 1, 1)
	tr.SetLive(map[string]Live{"dave": {Connections: 1}})
	tr.Forget(func(n string) bool { return n == "dave" })
	if err := tr.Stop(); err != nil {
		t.Fatal(err)
	}
	snap, _ := Read(dir)
	if snap.Running(time.Now()) || snap.Users["gone"] != nil || snap.Users["dave"].Connections != 0 {
		t.Fatalf("after stop = %+v", snap)
	}
}

func TestReadMissingFile(t *testing.T) {
	snap, err := Read(t.TempDir())
	if err != nil || snap.Users == nil || snap.Running(time.Now()) {
		t.Fatalf("snap = %+v err = %v", snap, err)
	}
}
