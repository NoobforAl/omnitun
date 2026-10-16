package tlsfiles

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/store"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-tls-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 40 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

// writeCert puts a self-signed certificate where certmagic would store the
// ACME certificate of domain.
func writeCert(t *testing.T, acmeDir, issuer, domain string) (certPath string) {
	t.Helper()
	dir := filepath.Join(acmeDir, "certificates", issuer, domain)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	certPath = filepath.Join(dir, domain+".crt")
	if err := store.WriteSelfSignedCert(certPath, filepath.Join(dir, domain+".key"), domain); err != nil {
		t.Fatal(err)
	}
	return certPath
}

func TestFallbackThenACMEThenRenewal(t *testing.T) {
	dir := tempDir(t)
	fallbackCert, fallbackKey := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	if err := store.WriteSelfSignedCert(fallbackCert, fallbackKey, "vpn.example.com"); err != nil {
		t.Fatal(err)
	}
	acmeDir := filepath.Join(dir, "acme")
	src := New(acmeDir, "vpn.example.com", fallbackCert, fallbackKey)

	// Before the first issuance the self-signed certificate is served.
	info, err := src.Info()
	if err != nil || info.Source != "self-signed" {
		t.Fatalf("before ACME: %+v err %v", info, err)
	}
	if cert, err := src.GetCertificate(nil); err != nil || cert == nil {
		t.Fatalf("GetCertificate: %v", err)
	}

	// Once the core has obtained the certificate it takes over, no restart.
	writeCert(t, acmeDir, "acme-v02.api.letsencrypt.org-directory", "vpn.example.com")
	if info, err = src.Info(); err != nil || info.Source != "acme" || info.Domain != "vpn.example.com" {
		t.Fatalf("after ACME: %+v err %v", info, err)
	}

	// A renewal (new file content) is picked up too.
	before := info.NotAfter
	time.Sleep(20 * time.Millisecond)
	path := writeCert(t, acmeDir, "acme-v02.api.letsencrypt.org-directory", "vpn.example.com")
	later := time.Now().Add(time.Second)
	os.Chtimes(path, later, later)
	if info, err = src.Info(); err != nil || !info.NotAfter.After(before.Add(-time.Hour)) {
		t.Fatalf("after renewal: %+v err %v", info, err)
	}
}

func TestOtherDomainsAreIgnored(t *testing.T) {
	dir := tempDir(t)
	fallbackCert, fallbackKey := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	store.WriteSelfSignedCert(fallbackCert, fallbackKey, "x")
	acmeDir := filepath.Join(dir, "acme")
	writeCert(t, acmeDir, "issuer", "other.example.com")
	src := New(acmeDir, "vpn.example.com", fallbackCert, fallbackKey)
	if info, err := src.Info(); err != nil || info.Source != "self-signed" {
		t.Fatalf("info = %+v err %v", info, err)
	}
	// No domain at all: always the fallback.
	if info, err := New(acmeDir, "", fallbackCert, fallbackKey).Info(); err != nil || info.Source != "self-signed" {
		t.Fatalf("no domain: %+v err %v", info, err)
	}
	if _, err := New(acmeDir, "", filepath.Join(dir, "missing.crt"), fallbackKey).Info(); err == nil {
		t.Fatal("a missing certificate did not fail")
	}
}
