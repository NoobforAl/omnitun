package engine

import (
	"errors"
	"testing"

	"github.com/sagernet/sing-box/adapter"
)

// A server name that cannot be looked up is a DNS problem, whatever the resolver said about
// itself: "connection refused" from [::1]:53 is not the server refusing anything.
func TestFailedLookupIsADNSError(t *testing.T) {
	err := errors.New("lookup vpn.example.com: (exchange4: vpn.example.com.: read udp [::1]:60276->[::1]:53: read: connection refused)")
	if got := plainError(err); got != "the server name could not be resolved" {
		t.Errorf("plainError = %q", got)
	}
	if got := errorCode(err.Error()); got != ErrDNS {
		t.Errorf("errorCode = %q, want %q", got, ErrDNS)
	}
	if got := errorCode("dial tcp 1.2.3.4:443: connect: connection refused"); got != ErrRefused {
		t.Errorf("a refused connection is now %q", got)
	}
}

type dnsPlatform struct {
	adapter.PlatformInterface
	servers []string
}

func (p dnsPlatform) SystemDNS() []string { return p.servers }

// Server names are looked up through the network's own resolver; with none reported
// the engine still names one, because sing-box's "local" resolver cannot work on a phone.
func TestSystemDNSNeverLeavesLocalToAndroid(t *testing.T) {
	cases := []struct {
		name     string
		platform adapter.PlatformInterface
		want     string
	}{
		{"desktop keeps sing-box's own lookup", nil, ""},
		{"network resolver", dnsPlatform{servers: []string{"192.168.1.1", "8.8.8.8"}}, "udp://192.168.1.1"},
		{"ipv6 resolver", dnsPlatform{servers: []string{"2001:db8::1"}}, "udp://[2001:db8::1]"},
		{"none reported", dnsPlatform{}, fallbackLocalDNS},
	}
	for _, c := range cases {
		e := New(Config{Platform: c.platform})
		if got := e.systemDNS(); got != c.want {
			t.Errorf("%s: systemDNS = %q, want %q", c.name, got, c.want)
		}
	}
}

// Each reason a protocol can fail for gets its own wording, and the general kinds keep working.
func TestSpecificFailureReasons(t *testing.T) {
	cases := []struct{ raw, code string }{
		{"REALITY: reality verification failed", ErrReality},
		{"tls: failed to verify certificate: x509: certificate has expired or is not yet valid", ErrCertDate},
		{"x509: certificate signed by unknown authority", ErrCertUntrusted},
		{"x509: certificate is valid for a.example, not b.example", ErrCertName},
		{"x509: something else about the certificate", ErrTLS},
		{"HMAC verification failed", ErrTLSKeys},
		{"no handshake from Home within 15s", ErrNoHandshake},
		{"timeout: no recent network activity", ErrUDPBlocked},
		{"listen tcp 127.0.0.1:2080: bind: address already in use", ErrPortInUse},
		{"AUTH_FAILED", ErrAuth},
		{"dial tcp 1.2.3.4:443: i/o timeout", ErrTimeout},
	}
	for _, c := range cases {
		if got := errorCode(c.raw); got != c.code {
			t.Errorf("%q: code %q, want %q", c.raw, got, c.code)
		}
		// The reworded sentence maps back to the same code (the engine sends the sentence).
		if got := errorCode(plainError(errors.New(c.raw))); got != c.code {
			t.Errorf("%q: after rewording the code is %q, want %q", c.raw, got, c.code)
		}
	}
}
