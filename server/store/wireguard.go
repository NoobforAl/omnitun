package store

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/netip"
)

// WireGuardSubnet is where the server and its users get tunnel addresses: the
// server takes .1, users the following addresses.
const WireGuardSubnet = "10.30.0.0/16"

// WireGuard is the server's WireGuard identity.
type WireGuard struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// NewWireGuardKeys returns a base64 X25519 private and public key pair, as
// WireGuard expects them.
func NewWireGuardKeys() (priv, pub string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	enc := base64.StdEncoding
	return enc.EncodeToString(key.Bytes()), enc.EncodeToString(key.PublicKey().Bytes()), nil
}

// WireGuardServerAddress is the server's address inside the tunnel.
func WireGuardServerAddress() string {
	p := netip.MustParsePrefix(WireGuardSubnet)
	return fmt.Sprintf("%s/%d", p.Addr().Next(), p.Bits())
}

// nextWireGuardAddress returns the first free user address (as a host address,
// "10.30.0.2/32"); users already have theirs in st.
func nextWireGuardAddress(st *State) (string, error) {
	p := netip.MustParsePrefix(WireGuardSubnet)
	used := map[netip.Addr]bool{p.Addr().Next(): true} // the server
	for _, u := range st.Users {
		if pf, err := netip.ParsePrefix(u.WGAddress); err == nil {
			used[pf.Addr()] = true
		}
	}
	for a := p.Addr().Next().Next(); p.Contains(a); a = a.Next() {
		if !used[a] {
			return netip.PrefixFrom(a, 32).String(), nil
		}
	}
	return "", fmt.Errorf("no free WireGuard addresses left in %s", WireGuardSubnet)
}

// giveWireGuard assigns keys and an address to a user that has none.
func giveWireGuard(st *State, u *User) error {
	if u.WGPrivateKey == "" {
		var err error
		if u.WGPrivateKey, u.WGPublicKey, err = NewWireGuardKeys(); err != nil {
			return err
		}
	}
	if u.WGAddress == "" {
		addr, err := nextWireGuardAddress(st)
		if err != nil {
			return err
		}
		u.WGAddress = addr
	}
	return nil
}
