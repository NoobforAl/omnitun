package engine

import (
	"context"
	"errors"
	"strings"
)

// Error codes reported in State.ErrorCode, so an app can word a failure in
// its own language. State.Error keeps the English sentence for apps (and
// logs) that do not translate.
const (
	ErrNoNetwork          = "NO_NETWORK"
	ErrTimeout            = "TIMEOUT"
	ErrRefused            = "REFUSED"
	ErrReset              = "RESET"
	ErrNoRoute            = "NO_ROUTE"
	ErrNetworkUnreachable = "NETWORK_UNREACHABLE"
	ErrDNS                = "DNS_FAILED"
	ErrTLS                = "TLS_ERROR"
	ErrAuth               = "AUTH_FAILED"
	ErrClosed             = "CLOSED"
	ErrTorTimeout         = "TOR_TIMEOUT"
	ErrTorBridgeTimeout   = "TOR_BRIDGE_TIMEOUT"
	ErrTorBadBridge       = "TOR_BAD_BRIDGE"
	ErrTorMissing         = "TOR_MISSING"
)

// errorKind ties a code to the raw error text it is recognised by and to the
// sentence the user gets. One table, so the code and the sentence agree.
type errorKind struct {
	code, match, text string
}

var errorKinds = [...]errorKind{
	{ErrTimeout, "context deadline exceeded", "the server did not answer in time"},
	{ErrTimeout, "i/o timeout", "the server did not answer in time"},
	{ErrRefused, "connection refused", "the server refused the connection"},
	{ErrReset, "connection reset", "the connection was reset by the network"},
	{ErrNoRoute, "no route to host", "the server cannot be reached from this network"},
	{ErrNetworkUnreachable, "network is unreachable", "the network is unreachable"},
	{ErrDNS, "no such host", "the server name could not be resolved"},
	{ErrTLS, "certificate", "the server's security certificate was rejected"},
	{ErrAuth, "authentication failed", "the username or password was rejected"},
	{ErrClosed, "EOF", "the server closed the connection"},
}

// Sentences of the Tor failures (torErrorText) and the offline message.
const (
	msgNoInternet        = "No internet connection"
	msgTorTimeoutBridge  = "Tor did not finish connecting in time. The bridge may be blocked or overloaded; try another bridge type (Snowflake or obfs4)"
	msgTorTimeout        = "Tor did not finish connecting in time. This network may block Tor; use a bridge (Snowflake or obfs4)"
	msgTorBadBridge      = "a Tor bridge line is not valid (it needs a transport, address and fingerprint)"
	msgTorClientNotBuilt = "this build does not include the Tor client"
)

// plainError words an error for the user: raw network and library messages
// become short sentences. The log keeps the original text.
func plainError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "the server did not answer in time"
	}
	msg := err.Error()
	for _, k := range errorKinds {
		if strings.Contains(msg, k.match) {
			return k.text
		}
	}
	return msg
}

// errNoFingerprint marks a Tor bridge line without the relay fingerprint that
// the Tor client (Arti) requires.
var errNoFingerprint = errors.New("a Tor bridge line needs the bridge's fingerprint (40 hex characters) after its address; copy a complete line from bridges.torproject.org")

// torErrorText words a Tor failure for the user.
func torErrorText(err error, bridges bool) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		if bridges {
			return msgTorTimeoutBridge
		}
		return msgTorTimeout
	case strings.Contains(err.Error(), "bad bridge line"):
		return msgTorBadBridge
	case strings.Contains(err.Error(), "the Tor client (omnitun-tor) is not installed"):
		return msgTorClientNotBuilt
	}
	return plainError(err)
}

// errorCode names the kind of failure a state's English message describes,
// or "" when it is something unusual (the app then shows the message as it is).
func errorCode(message string) string {
	switch {
	case message == "":
		return ""
	case strings.Contains(message, msgNoInternet):
		return ErrNoNetwork
	case strings.Contains(message, msgTorTimeoutBridge):
		return ErrTorBridgeTimeout
	case strings.Contains(message, msgTorTimeout):
		return ErrTorTimeout
	case strings.Contains(message, msgTorBadBridge):
		return ErrTorBadBridge
	case strings.Contains(message, msgTorClientNotBuilt):
		return ErrTorMissing
	}
	for _, k := range errorKinds {
		if strings.Contains(message, k.text) {
			return k.code
		}
	}
	// A raw error that was not reworded (some stages report err.Error() as is).
	for _, k := range errorKinds {
		if strings.Contains(message, k.match) {
			return k.code
		}
	}
	return ""
}
