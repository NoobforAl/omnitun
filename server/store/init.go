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
	"time"
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
}

// DefaultPorts per protocol. TCP and UDP protocols may share a number.
var DefaultPorts = map[string]int{
	VLESSReality: 443,
	Shadowsocks:  8388,
	Trojan:       8443,
	Hysteria2:    8443, // UDP
	TUIC:         8444, // UDP
	OpenVPNUDP:   1194,
	OpenVPNTCP:   1194,
}

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
	if st.Server.OpenVPN, err = NewOpenVPN(opts.Name); err != nil {
		return nil, err
	}
	for _, p := range AllProtocols {
		port := DefaultPorts[p]
		if v, ok := opts.Ports[p]; ok {
			port = v
		}
		st.Server.Protocols[p] = &Proto{Enabled: true, Port: port}
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

// WriteSelfSignedCert writes a 10-year ECDSA certificate for host.
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
		NotAfter:     time.Now().AddDate(10, 0, 0),
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
