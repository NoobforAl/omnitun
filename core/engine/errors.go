package engine

import (
	"context"
	"errors"
	"strings"
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
	for _, r := range [...]struct{ match, text string }{
		{"context deadline exceeded", "the server did not answer in time"},
		{"i/o timeout", "the server did not answer in time"},
		{"connection refused", "the server refused the connection"},
		{"connection reset", "the connection was reset by the network"},
		{"no route to host", "the server cannot be reached from this network"},
		{"network is unreachable", "the network is unreachable"},
		{"no such host", "the server name could not be resolved"},
		{"certificate", "the server's security certificate was rejected"},
		{"authentication failed", "the username or password was rejected"},
		{"EOF", "the server closed the connection"},
	} {
		if strings.Contains(msg, r.match) {
			return r.text
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
			return "Tor did not finish connecting in time. The bridge may be blocked or overloaded; try another bridge type (Snowflake or obfs4)"
		}
		return "Tor did not finish connecting in time. This network may block Tor; use a bridge (Snowflake or obfs4)"
	case strings.Contains(err.Error(), "bad bridge line"):
		return "a Tor bridge line is not valid (it needs a transport, address and fingerprint)"
	case strings.Contains(err.Error(), "the Tor client (omnitun-tor) is not installed"):
		return "this build does not include the Tor client"
	}
	return plainError(err)
}
