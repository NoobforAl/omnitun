package store

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"strings"
	"time"
)

// OpenVPN holds the server's OpenVPN key material (PEM). Clients authenticate
// with their username and password; the CA only proves the server's identity.
type OpenVPN struct {
	CA         string `json:"ca"`
	ServerCert string `json:"server_cert"`
	ServerKey  string `json:"server_key"`
	// TLSCrypt is the shared tls-crypt key: it hides the OpenVPN handshake
	// and drops packets from anyone without the key.
	TLSCrypt string `json:"tls_crypt"`
}

// NewOpenVPN generates a CA, a server certificate and a tls-crypt key.
func NewOpenVPN(name string) (OpenVPN, error) {
	pemOf := func(typ string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
	}
	serial := func() *big.Int {
		n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
		return n
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return OpenVPN{}, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber: serial(), Subject: pkix.Name{CommonName: name + " OpenVPN CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(20, 0, 0),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return OpenVPN{}, err
	}
	ca, _ := x509.ParseCertificate(caDER)
	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return OpenVPN{}, err
	}
	srvTmpl := &x509.Certificate{
		SerialNumber: serial(), Subject: pkix.Name{CommonName: "server"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(10, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	srvDER, err := x509.CreateCertificate(rand.Reader, srvTmpl, ca, &srvKey.PublicKey, caKey)
	if err != nil {
		return OpenVPN{}, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(srvKey)
	if err != nil {
		return OpenVPN{}, err
	}
	return OpenVPN{
		CA:         pemOf("CERTIFICATE", caDER),
		ServerCert: pemOf("CERTIFICATE", srvDER),
		ServerKey:  pemOf("PRIVATE KEY", keyDER),
		TLSCrypt:   staticKey(),
	}, nil
}

// staticKey returns a 2048-bit OpenVPN static key (the tls-crypt file format).
func staticKey() string {
	raw := make([]byte, 256)
	rand.Read(raw)
	h := hex.EncodeToString(raw)
	var b strings.Builder
	b.WriteString("-----BEGIN OpenVPN Static key V1-----\n")
	for i := 0; i < len(h); i += 32 {
		b.WriteString(h[i:i+32] + "\n")
	}
	b.WriteString("-----END OpenVPN Static key V1-----\n")
	return b.String()
}

// migrate fills in settings added after the state was created. It reports
// whether anything changed. New protocols start disabled on existing servers,
// so an upgrade never opens ports by itself.
func (st *State) migrate() (bool, error) {
	changed := false
	if st.Server.OpenVPN.CA == "" {
		o, err := NewOpenVPN(st.Server.Name)
		if err != nil {
			return false, err
		}
		st.Server.OpenVPN, changed = o, true
	}
	if st.Server.Protocols == nil {
		st.Server.Protocols = map[string]*Proto{}
	}
	for _, p := range AllProtocols {
		if st.Server.Protocols[p] == nil {
			st.Server.Protocols[p] = &Proto{Enabled: false, Port: DefaultPorts[p]}
			changed = true
		}
	}
	if st.Server.WireGuard.PrivateKey == "" {
		priv, pub, err := NewWireGuardKeys()
		if err != nil {
			return false, err
		}
		st.Server.WireGuard, changed = WireGuard{PrivateKey: priv, PublicKey: pub}, true
	}
	for _, u := range st.Users {
		if u.WGPrivateKey == "" || u.WGAddress == "" {
			if err := giveWireGuard(st, u); err != nil {
				return false, err
			}
			changed = true
		}
	}
	if st.Server.IKEv2PSK == "" {
		st.Server.IKEv2PSK, changed = RandomHex(16), true
	}
	if st.Server.API.Token == "" {
		st.Server.API = API{Listen: "127.0.0.1", Port: DefaultAPIPort, Token: RandomHex(24)}
		changed = true
	}
	return changed, nil
}
