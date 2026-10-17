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
	ErrCertDate           = "CERT_DATE"
	ErrCertUntrusted      = "CERT_UNTRUSTED"
	ErrCertName           = "CERT_NAME"
	ErrTLSKeys            = "TLS_KEYS"
	ErrReality            = "REALITY_REJECTED"
	ErrNoHandshake        = "NO_HANDSHAKE"
	ErrUDPBlocked         = "UDP_BLOCKED"
	ErrPortInUse          = "PORT_IN_USE"
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
	// First: a failed name lookup also says "connection refused" or "i/o timeout" (about the
	// resolver), which would otherwise be blamed on the server.
	{ErrDNS, "lookup ", "the server name could not be resolved"},
	// A protocol can fail for several reasons that need different fixes, so the specific
	// ones come before the general kind they belong to.
	{ErrReality, "reality verification failed", "the server rejected the Reality handshake (check the public key, short ID and server name)"},
	{ErrReality, "REALITY", "the server rejected the Reality handshake (check the public key, short ID and server name)"},
	{ErrCertDate, "certificate has expired", "the server's certificate has expired or is not valid yet (also check this phone's date and time)"},
	{ErrCertDate, "not yet valid", "the server's certificate has expired or is not valid yet (also check this phone's date and time)"},
	{ErrCertUntrusted, "unknown authority", "the server's certificate is not signed by a trusted authority"},
	{ErrCertName, "certificate is valid for", "the server's certificate is for a different name than the one you connect to"},
	{ErrCertName, "certificate is not valid for", "the server's certificate is for a different name than the one you connect to"},
	{ErrTLSKeys, "tls-crypt", "the shared key (tls-auth or tls-crypt) does not match the server's"},
	{ErrTLSKeys, "tls-auth", "the shared key (tls-auth or tls-crypt) does not match the server's"},
	{ErrTLSKeys, "HMAC verification failed", "the shared key (tls-auth or tls-crypt) does not match the server's"},
	{ErrNoHandshake, "no handshake from", "the server did not answer the handshake (check the keys, the endpoint and that its UDP port is open)"},
	{ErrUDPBlocked, "no recent network activity", "the server did not answer over UDP; this network may block it, try a TCP-based server"},
	{ErrPortInUse, "address already in use", "the local port is already used by another app"},
	{ErrTimeout, "context deadline exceeded", "the server did not answer in time"},
	{ErrTimeout, "i/o timeout", "the server did not answer in time"},
	{ErrRefused, "connection refused", "the server refused the connection"},
	{ErrReset, "connection reset", "the connection was reset by the network"},
	{ErrNoRoute, "no route to host", "the server cannot be reached from this network"},
	{ErrNetworkUnreachable, "network is unreachable", "the network is unreachable"},
	{ErrDNS, "no such host", "the server name could not be resolved"},
	{ErrTLS, "certificate", "the server's security certificate was rejected"},
	{ErrAuth, "authentication failed", "the username or password was rejected"},
	{ErrAuth, "AUTH_FAILED", "the username or password was rejected"},
	{ErrAuth, "401 Unauthorized", "the username or password was rejected"},
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
