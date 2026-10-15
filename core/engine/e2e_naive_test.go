//go:build with_quic && with_utls && with_wireguard && with_gvisor && with_naive_outbound

// Run with the extra tags, e.g. on Windows (cronet loads libcronet.dll):
//   go test -tags=with_gvisor,with_quic,with_wireguard,with_utls,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0 -ldflags=-checklinkname=0 -run Naive ./core/engine/

package engine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/importer"
)

// naiveServerCert issues a certificate for naive.test from a fresh CA and
// returns the CA (to pin) and the server certificate and key.
func naiveServerCert(t *testing.T) (ca, cert, key string) {
	t.Helper()
	pemOf := func(typ string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
	}
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Naive Test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caDER)
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "naive.test"},
		DNSNames: []string{"naive.test"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &k.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalPKCS8PrivateKey(k)
	return pemOf("CERTIFICATE", caDER), pemOf("CERTIFICATE", der), pemOf("PRIVATE KEY", keyDER)
}

// useCronetDLL puts the libcronet.dll shipped in the Go module cache on PATH.
func useCronetDLL(t *testing.T) {
	if runtime.GOOS != "windows" {
		return
	}
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skip("go env GOMODCACHE:", err)
	}
	dirs, _ := filepath.Glob(filepath.Join(strings.TrimSpace(string(out)), "github.com", "sagernet", "cronet-go", "lib", "windows_"+runtime.GOARCH+"@*"))
	if len(dirs) == 0 {
		t.Skip("libcronet.dll not in the module cache")
	}
	sort.Strings(dirs)
	t.Setenv("PATH", dirs[len(dirs)-1]+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestEndToEndNaive(t *testing.T) {
	useCronetDLL(t)
	ca, cert, key := naiveServerCert(t)
	port := freePort(t)
	startServer(t, map[string]any{
		"type": "naive", "listen": "127.0.0.1", "listen_port": port,
		"users": []any{map[string]any{"username": "nu", "password": "np"}},
		"tls":   map[string]any{"enabled": true, "certificate": cert, "key": key},
	})
	// As served by omnitun-server's sing-box subscription: the certificate is pinned.
	doc, _ := json.Marshal(map[string]any{"outbounds": []any{map[string]any{
		"type": "naive", "tag": "naive", "server": "127.0.0.1", "server_port": port,
		"username": "nu", "password": "np",
		"tls": map[string]any{"enabled": true, "server_name": "naive.test", "certificate": ca},
	}}})
	checkThrough(t, string(doc))

	// The share link form parses to the same connection settings.
	res, err := importer.ParseLink(fmt.Sprintf("naive+https://nu:np@127.0.0.1:%d/?sni=naive.test#n", port))
	if err != nil || res.Username != "nu" || res.TLS.ServerName != "naive.test" {
		t.Fatalf("link: %v %+v", err, res)
	}
}
