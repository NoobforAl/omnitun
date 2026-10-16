// Package stats keeps per-user traffic counters and live connection numbers.
// The running server updates a Tracker and saves it to stats.json in the data
// directory; the CLI, the TUI and the web API read that file, so they show
// the same numbers even though they run in other processes.
package stats

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const fileName = "stats.json"

// GiB is the unit of a user's quota ("GB" in the CLI and clients).
const GiB = 1 << 30

// User is one account's counters. Bytes are as seen by the server: Up is
// from the client, Down is to the client.
type User struct {
	Up   int64 `json:"up"`   // lifetime
	Down int64 `json:"down"` // lifetime

	// Month is the calendar month (UTC, "2026-10") the Month* counters
	// belong to; they restart when it changes or when usage is reset.
	Month     string `json:"month"`
	MonthUp   int64  `json:"month_up"`
	MonthDown int64  `json:"month_down"`

	// ResetApplied is the last usage-reset request (store.User.UsageResetAt)
	// that was applied to the counters.
	ResetApplied time.Time `json:"reset_applied,omitzero"`

	// Live numbers, refreshed every tick while the server runs.
	Connections int       `json:"connections"`
	UpBps       int64     `json:"up_bps"`
	DownBps     int64     `json:"down_bps"`
	LastSeen    time.Time `json:"last_seen,omitzero"`
}

// MonthTotal is the traffic counted against the quota this month.
func (u User) MonthTotal() int64 { return u.MonthUp + u.MonthDown }

// Snapshot is the stats file.
type Snapshot struct {
	// UpdatedAt is when the server last wrote the file; a stale value means
	// the server is not running.
	UpdatedAt time.Time        `json:"updated_at"`
	StartedAt time.Time        `json:"started_at"`
	Users     map[string]*User `json:"users"`
}

// Running reports whether the server wrote the file recently.
func (s Snapshot) Running(now time.Time) bool {
	return !s.UpdatedAt.IsZero() && now.Sub(s.UpdatedAt) < 20*time.Second
}

// Read loads the stats file from dir; a missing file is an empty snapshot.
func Read(dir string) (Snapshot, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{Users: map[string]*User{}}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, err
	}
	if s.Users == nil {
		s.Users = map[string]*User{}
	}
	return s, nil
}

// Tracker is the live, in-memory counters of a running server.
type Tracker struct {
	dir string
	now func() time.Time

	mu      sync.Mutex
	snap    Snapshot
	stopped bool
}

// NewTracker loads the counters saved in dir (so they survive restarts).
func NewTracker(dir string) *Tracker {
	t := &Tracker{dir: dir, now: time.Now}
	t.snap, _ = Read(dir)
	t.snap.StartedAt = t.now().UTC()
	return t
}

func monthKey(t time.Time) string { return t.UTC().Format("2006-01") }

func (t *Tracker) user(name string, now time.Time) *User {
	u := t.snap.Users[name]
	if u == nil {
		u = &User{}
		t.snap.Users[name] = u
	}
	if m := monthKey(now); u.Month != m {
		u.Month, u.MonthUp, u.MonthDown = m, 0, 0
	}
	return u
}

// Add counts traffic for a user.
func (t *Tracker) Add(name string, up, down int64) {
	if name == "" || (up == 0 && down == 0) {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	u := t.user(name, t.now())
	u.Up += up
	u.Down += down
	u.MonthUp += up
	u.MonthDown += down
}

// Live is a user's current connections and speed.
type Live struct {
	Connections int
	UpBps       int64
	DownBps     int64
}

// SetLive replaces the live numbers of every user: users missing from live
// have no connections.
func (t *Tracker) SetLive(live map[string]Live) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	for name, u := range t.snap.Users {
		l := live[name]
		u.Connections, u.UpBps, u.DownBps = l.Connections, l.UpBps, l.DownBps
	}
	for name, l := range live {
		u := t.user(name, now)
		u.Connections, u.UpBps, u.DownBps = l.Connections, l.UpBps, l.DownBps
		if l.Connections > 0 {
			u.LastSeen = now.UTC()
		}
	}
}

// ApplyReset zeroes a user's monthly counters when a reset was requested after
// the last applied one. It reports whether it did.
func (t *Tracker) ApplyReset(name string, requested time.Time) bool {
	if requested.IsZero() {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	u := t.user(name, t.now())
	if !requested.After(u.ResetApplied) {
		return false
	}
	u.ResetApplied = requested
	u.MonthUp, u.MonthDown = 0, 0
	return true
}

// MonthTotal is a user's traffic this month.
func (t *Tracker) MonthTotal(name string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.user(name, t.now()).MonthTotal()
}

// Forget drops users that no longer exist.
func (t *Tracker) Forget(keep func(name string) bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for name := range t.snap.Users {
		if !keep(name) {
			delete(t.snap.Users, name)
		}
	}
}

// Save writes the stats file atomically.
func (t *Tracker) Save() error {
	t.mu.Lock()
	t.snap.UpdatedAt = t.now().UTC()
	if t.stopped {
		t.snap.UpdatedAt = time.Time{} // readers see at once that the server is down
	}
	data, err := json.MarshalIndent(t.snap, "", "  ")
	t.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(t.dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(t.dir, fileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	for attempt := 0; ; attempt++ {
		err = os.Rename(tmp, path)
		if err == nil || attempt == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Stop marks the server as stopped (no live connections) and saves.
func (t *Tracker) Stop() error {
	t.mu.Lock()
	t.stopped = true
	for _, u := range t.snap.Users {
		u.Connections, u.UpBps, u.DownBps = 0, 0, 0
	}
	t.mu.Unlock()
	return t.Save()
}
