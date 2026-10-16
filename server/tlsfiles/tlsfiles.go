// Package tlsfiles finds the certificate the server's own HTTPS endpoints
// (subscriptions, web API) should present: the ACME certificate for the
// domain once the VPN core has obtained it (certmagic keeps it under
// <acme dir>/certificates/<issuer>/<domain>/), and until then the
// self-signed one from the data directory.
package tlsfiles

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Source supplies certificates, picking up renewals without a restart.
type Source struct {
	acmeDir, domain           string
	fallbackCert, fallbackKey string

	mu     sync.Mutex
	loaded loadedCert
}

type loadedCert struct {
	cert     *tls.Certificate
	leaf     *x509.Certificate
	acme     bool
	certPath string
	keyPath  string
	modTime  time.Time
	checked  time.Time
}

// New returns a source. domain may be empty (no ACME): only the fallback
// certificate files are used.
func New(acmeDir, domain, fallbackCert, fallbackKey string) *Source {
	return &Source{acmeDir: acmeDir, domain: domain, fallbackCert: fallbackCert, fallbackKey: fallbackKey}
}

// Info describes the certificate in use.
type Info struct {
	// Source is "acme" or "self-signed".
	Source    string    `json:"source"`
	Domain    string    `json:"domain,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
}

// acmePair is the newest valid ACME certificate and key paths for the domain.
func (s *Source) acmePair() (certPath, keyPath string, ok bool) {
	if s.domain == "" {
		return "", "", false
	}
	matches, _ := filepath.Glob(filepath.Join(s.acmeDir, "certificates", "*", s.domain, s.domain+".crt"))
	var best time.Time
	for _, m := range matches {
		cert, err := readLeaf(m)
		if err != nil || !cert.NotAfter.After(time.Now()) {
			continue
		}
		if cert.NotAfter.After(best) {
			best, certPath = cert.NotAfter, m
		}
	}
	if certPath == "" {
		return "", "", false
	}
	return certPath, certPath[:len(certPath)-len(".crt")] + ".key", true
}

func readLeaf(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, errors.New("no certificate in " + path)
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
}

// current returns the certificate to serve, reloading when the files changed
// (or at most once a minute, to notice a new ACME certificate).
func (s *Source) current() (*loadedCert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	certPath, keyPath, isACME := s.acmePair()
	if !isACME {
		certPath, keyPath = s.fallbackCert, s.fallbackKey
	}
	fi, err := os.Stat(certPath)
	if err != nil {
		return nil, err
	}
	l := s.loaded
	if l.cert != nil && l.certPath == certPath && l.modTime.Equal(fi.ModTime()) && l.acme == isACME {
		return &s.loaded, nil
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}
	s.loaded = loadedCert{cert: &cert, leaf: leaf, acme: isACME, certPath: certPath, keyPath: keyPath, modTime: fi.ModTime(), checked: time.Now()}
	return &s.loaded, nil
}

// GetCertificate is a tls.Config.GetCertificate.
func (s *Source) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	l, err := s.current()
	if err != nil {
		return nil, err
	}
	return l.cert, nil
}

// Paths returns the certificate and key files currently served.
func (s *Source) Paths() (cert, key string, err error) {
	l, err := s.current()
	if err != nil {
		return "", "", err
	}
	return l.certPath, l.keyPath, nil
}

// PEM returns the certificate chain and private key currently served, for
// servers that take PEM bytes (SSTP). They are read when called, so a renewed
// certificate is used the next time such a server is started.
func (s *Source) PEM() (cert, key []byte, err error) {
	l, err := s.current()
	if err != nil {
		return nil, nil, err
	}
	if cert, err = os.ReadFile(l.certPath); err != nil {
		return nil, nil, err
	}
	if key, err = os.ReadFile(l.keyPath); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// TLSConfig returns a server TLS configuration using the source.
func (s *Source) TLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: s.GetCertificate}
}

// Info reports the certificate currently served.
func (s *Source) Info() (Info, error) {
	l, err := s.current()
	if err != nil {
		return Info{}, err
	}
	info := Info{Source: "self-signed", Issuer: l.leaf.Issuer.CommonName, NotBefore: l.leaf.NotBefore, NotAfter: l.leaf.NotAfter}
	if l.acme {
		info.Source, info.Domain = "acme", s.domain
	}
	return info, nil
}
