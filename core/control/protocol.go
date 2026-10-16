// Package control is the UI <-> core protocol: newline-delimited JSON over a
// loopback TCP connection, authenticated with a shared token.
//
// Client -> server:  {"id":1,"method":"start","params":{...}}
// Server -> client:  {"id":1,"result":{...}}  or  {"id":1,"error":{"code":"...","message":"..."}}
// Server push:       {"event":"state","data":{...}}  /  {"event":"log","data":{...}}
//
// The first request on a connection must be "hello" with the token.
package control

import (
	"encoding/json"
	"time"

	"github.com/NoobforAl/omnitun/core/profile"
)

// Version of the control protocol.
const Version = 1

// Request from the UI.
type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response to a request.
type Response struct {
	ID     int64  `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Event pushed by the server.
type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// Error codes shown to the user through translated messages.
const (
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnknownMethod      = "UNKNOWN_METHOD"
	CodeImportFailed       = "IMPORT_FAILED"
	CodeStartFailed        = "START_FAILED"
	CodeNoNetwork          = "NO_NETWORK"
	CodeNotRunning         = "NOT_RUNNING"
	CodeSubscriptionFailed = "SUBSCRIPTION_FAILED"
	CodeAuthFailed         = "AUTH_FAILED"
)

// Error is a typed failure.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// HelloParams authenticates a connection.
type HelloParams struct {
	Token string `json:"token"`
}

// ImportParams carries text to parse.
type ImportParams struct {
	Text string `json:"text"`
}

// BuildProfileParams creates a profile from an editor's values, or edits
// Profile when it is given (fields the editor does not show are kept).
type BuildProfileParams struct {
	Protocol string            `json:"protocol,omitempty"`
	Values   map[string]string `json:"values"`
	Profile  *profile.Profile  `json:"profile,omitempty"`
}

// ProfileParams names one profile (for its editor values or share link).
type ProfileParams struct {
	Profile *profile.Profile `json:"profile"`
}

// StartParams starts the tunnel.
type StartParams struct {
	Profiles     []*profile.Profile `json:"profiles"`
	Mode         string             `json:"mode,omitempty"` // "proxy" or "tun"
	ListenPort   int                `json:"listen_port,omitempty"`
	Selected     string             `json:"selected,omitempty"` // profile ID or "auto"
	ProxyPrivate bool               `json:"proxy_private,omitempty"`
	SystemProxy  bool               `json:"system_proxy,omitempty"`
	DNS          string             `json:"dns,omitempty"` // remote resolver URL, see singbox.Options.DNS
	// Failover switches to the next working profile (in list order) when the selected one fails.
	Failover bool   `json:"failover,omitempty"`
	TestURL  string `json:"test_url,omitempty"`
	// Android per-app split tunneling (package names); include wins over exclude.
	IncludePackages []string `json:"include_packages,omitempty"`
	ExcludePackages []string `json:"exclude_packages,omitempty"`
}

// SelectParams switches the active profile.
type SelectParams struct {
	ID string `json:"id"`
}

// TestParams measures latency of profiles.
type TestParams struct {
	Profiles []*profile.Profile `json:"profiles"`
	URL      string             `json:"url,omitempty"`
}

// LogEntry is pushed as a "log" event.
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// ControlFile is written by the daemon so clients can find and authenticate to it.
type ControlFile struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}
