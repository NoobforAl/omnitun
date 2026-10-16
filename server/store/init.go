package store

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"slices"
	"time"

	"github.com/NoobforAl/omnitun/core/protocols"
)

// InitOptions configures a new server.
type InitOptions struct {
	Name       string
	PublicHost string
	// Ports overrides default ports per protocol.
	Ports map[string]int
	// RealityHandshake is the site Reality impersonates (default www.microsoft.com:443).
	RealityHandshake string
	SubPort          int
	// Enabled lists the protocols to switch on; empty enables all of them.
	Enabled []string
}

// EnabledAtInit decides whether a protocol starts enabled: those asked for,
// or by default every protocol that needs nothing special from the host.
func EnabledAtInit(key string, asked []string) bool {
	if len(asked) > 0 {
		return slices.Contains(asked, key)
	}
	if p, ok := protocols.Get(key); ok && p.Spec().DefaultOff {
		return false
	}
	return true
}

// DefaultEnabled lists the protocols a new server enables when none are asked for.
func DefaultEnabled() []string {
	var out []string
	for _, k := range AllProtocols {
		if EnabledAtInit(k, nil) {
			out = append(out, k)
		}
	}
	return out
}

// DefaultPorts per protocol, from the protocol registry. TCP and UDP
// protocols may share a number.
var DefaultPorts = protocols.DefaultPorts()

// Init creates a state with fresh keys and a self-signed certificate.
func Init(s *Store, opts InitOptions) (*State, error) {
	if s.Exists() {
		return nil, fmt.Errorf("already initialized in %s", s.Dir)
	}
	if opts.PublicHost == "" {
		return nil, fmt.Errorf("public host (IP or domain) is required")
	}
	if opts.Name == "" {
		opts.Name = "Omnitun"
	}
	if opts.RealityHandshake == "" {
		opts.RealityHandshake = "www.microsoft.com:443"
	}
	if opts.SubPort == 0 {
		opts.SubPort = 2096
	}
	priv, pub, err := RealityKeyPair()
	if err != nil {
		return nil, err
	}
	st := &State{
		Version: 1,
		Server: Server{
			Name:        opts.Name,
			PublicHost:  opts.PublicHost,
			Protocols:   map[string]*Proto{},
			Reality:     Reality{PrivateKey: priv, PublicKey: pub, ShortID: RandomHex(4), Handshake: opts.RealityHandshake},
			SSServerKey: RandomBase64(16),
			CertFile:    "tls.crt",
			KeyFile:     "tls.key",
			SelfSigned:  true,
			Subscription: Subscription{
				Enabled: true, Listen: "0.0.0.0", Port: opts.SubPort, Path: RandomHex(8),
			},
		},
	}
	st.Server.API = API{Listen: "127.0.0.1", Port: DefaultAPIPort, Token: RandomHex(24)}
	wgPriv, wgPub, err := NewWireGuardKeys()
	if err != nil {
		return nil, err
	}
	st.Server.WireGuard = WireGuard{PrivateKey: wgPriv, PublicKey: wgPub}
	if st.Server.AmneziaWG, err = NewAmneziaWG(); err != nil {
		return nil, err
	}
	if st.Server.OpenVPN, err = NewOpenVPN(opts.Name); err != nil {
		return nil, err
	}
	for _, p := range AllProtocols {
		port := DefaultPorts[p]
		if v, ok := opts.Ports[p]; ok {
			port = v
		}
		st.Server.Protocols[p] = &Proto{Enabled: EnabledAtInit(p, opts.Enabled), Port: port}
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	if err := WriteSelfSignedCert(s.Path(st.Server.CertFile), s.Path(st.Server.KeyFile), opts.PublicHost); err != nil {
		return nil, err
	}
	if err := s.Save(st); err != nil {
		return nil, err
	}
	return st, nil
}

// RealityKeyPair returns an X25519 key pair encoded as Reality expects.
func RealityKeyPair() (priv, pub string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(key.Bytes()), enc.EncodeToString(key.PublicKey().Bytes()), nil
}

// RandomBase64 returns n random bytes in standard base64 (Shadowsocks-2022 keys).
func RandomBase64(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// SelfSignedValidity is short because Chromium (and so NaiveProxy, which pins
// this certificate) rejects long-lived certificates even when trusted
// explicitly, following the CA/Browser Forum limits (200 days in 2026,
// 100 in 2027). The server renews it on start well before it expires.
const SelfSignedValidity = 90 * 24 * time.Hour

// RenewSelfSignedCert replaces the server's self-signed certificate when it
// expires within 30 days or was issued for longer than SelfSignedValidity. It
// reports whether it renewed the certificate.
func RenewSelfSignedCert(s *Store, st *State) (bool, error) {
	if !st.Server.SelfSigned {
		return false, nil
	}
	certPath := s.Path(st.Server.CertFile)
	if data, err := os.ReadFile(certPath); err == nil {
		if block, _ := pem.Decode(data); block != nil {
			if cert, err := x509.ParseCertificate(block.Bytes); err == nil &&
				time.Until(cert.NotAfter) > 30*24*time.Hour && cert.NotAfter.Sub(cert.NotBefore) <= SelfSignedValidity+24*time.Hour {
				return false, nil
			}
		}
	}
	return true, WriteSelfSignedCert(certPath, s.Path(st.Server.KeyFile), st.Server.PublicHost)
}

// WriteSelfSignedCert writes a self-signed ECDSA certificate for host, valid
// for SelfSignedValidity.
func WriteSelfSignedCert(certPath, keyPath, host string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(SelfSignedValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return err
	}
	return os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)
}
