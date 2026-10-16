package protocols

import (
	"fmt"

	"github.com/NoobforAl/omnitun/core/profile"
)

// Default tunnel networks of the host-level servers (NAT must cover them).
const (
	SSTPPool  = "10.9.0.0/24"
	IKEv2Pool = "10.10.10.0/24"
)

// PortUse is one port a protocol listens on.
type PortUse struct {
	Port  int
	Proto string // "tcp" or "udp"
}

// PortProtocol is implemented by protocols that listen on more ports than
// their configured one (IKEv2 also needs UDP 500).
type PortProtocol interface {
	// Listens returns every port the protocol uses, given its main port.
	Listens(port int) []PortUse
}

// hostBase is shared by the protocols served by a process-level server (a
// TUN device on the host plus NAT) instead of by sing-box.
type hostBase struct{}

// Inbounds is empty: sing-box serves nothing for these protocols.
func (hostBase) Inbounds(Env, int, []Account) Config { return Config{} }

// ------------------------------------------------------------------- SSTP

type sstp struct{ hostBase }

func (sstp) Spec() Spec {
	return Spec{
		Key: "sstp", Name: "SSTP", Label: "SSTP", Profile: profile.SSTP,
		Description: "Windows' built-in VPN over TLS (MS-CHAPv2). Needs Linux, root/CAP_NET_ADMIN and a trusted certificate for Windows (use `cert acme`).",
		Transport:   "tcp", DefaultPort: 8448, NeedsCertificate: true, ShareLink: true,
		Host: true, DefaultOff: true,
	}
}

func (s sstp) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	if err := safeField("password", u.Password, unsafeInLines); err != nil {
		return nil, err
	}
	return &profile.Profile{
		Name: profileName(env, s.Spec()), Protocol: profile.SSTP, Server: env.Host, Port: port,
		Username: u.Name, Password: u.Password, TLS: serverTLS(env),
	}, nil
}

// ------------------------------------------------------------------ IKEv2

type ikev2 struct{ hostBase }

func (ikev2) Spec() Spec {
	return Spec{
		Key: "ikev2", Name: "IKEv2/IPsec", Label: "IKEv2", Profile: profile.IKEv2,
		Description: "IPsec with the server's certificate and per-user passwords (EAP-MSCHAPv2); also over TCP for networks that block UDP. Needs Linux and root/CAP_NET_ADMIN, and a trusted certificate for phones and Windows (use `cert acme`). Uses UDP 500 and 4500.",
		Transport:   "udp", DefaultPort: 4500, NeedsCertificate: true, ShareLink: true,
		Host: true, DefaultOff: true,
	}
}

// Listens: IKE on UDP 500, NAT-T on the main port, and IKE/ESP over TCP on it.
func (ikev2) Listens(port int) []PortUse {
	return []PortUse{{500, "udp"}, {port, "udp"}, {port, "tcp"}}
}

func (i ikev2) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	if err := safeField("password", u.Password, unsafeInLines); err != nil {
		return nil, err
	}
	// EAP identifies the user; the server proves itself with its certificate
	// (no secret shared between users). A self-signed certificate is pinned,
	// a real one is checked against the system's roots. TCP encapsulation
	// works on networks that block UDP.
	opts := &profile.IKEv2Opts{Auth: "eap", RemoteID: env.Host, TCP: true}
	if env.SelfSigned {
		if env.CertPEM == "" {
			return nil, fmt.Errorf("the server certificate is needed to pin it for IKEv2")
		}
		opts.CACert = env.CertPEM
	}
	return &profile.Profile{
		Name: profileName(env, i.Spec()), Protocol: profile.IKEv2, Server: env.Host, Port: port,
		Username: u.Name, Password: u.Password, IKEv2: opts,
	}, nil
}
