package store

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
)

// AmneziaWGSubnet is the AmneziaWG tunnel network: the server takes .1 and a
// user's address is their WireGuard address moved into this network (10.30.a.b
// becomes 10.31.a.b), so the two servers can run together.
const AmneziaWGSubnet = "10.31.0.0/16"

// AmneziaWG is the server's AmneziaWG identity and obfuscation settings.
type AmneziaWG struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	// Params are the obfuscation parameters (jc, jmin, jmax, s1, s2, h1-h4)
	// clients must use too.
	Params map[string]string `json:"params"`
}

// randUint returns a random integer in [lo, hi].
func randUint(lo, hi uint32) uint32 {
	var b [4]byte
	rand.Read(b[:])
	return lo + binary.BigEndian.Uint32(b[:])%(hi-lo+1)
}

// NewAmneziaWG makes a key pair and random obfuscation parameters that
// satisfy AmneziaWG's rules: jmin < jmax <= 1280, the init and response
// padding sizes differ (s2 != s1+56), and the four message-type headers are
// distinct and not WireGuard's own 1-4.
func NewAmneziaWG() (AmneziaWG, error) {
	priv, pub, err := NewWireGuardKeys()
	if err != nil {
		return AmneziaWG{}, err
	}
	jmin := randUint(40, 80)
	s1 := randUint(15, 120)
	s2 := randUint(15, 120)
	for s2 == s1+56 {
		s2 = randUint(15, 120)
	}
	hs := map[uint32]bool{}
	var h [4]uint32
	for i := range h {
		for {
			v := randUint(5, 0x7fffffff)
			if !hs[v] {
				hs[v], h[i] = true, v
				break
			}
		}
	}
	u := func(v uint32) string { return strconv.FormatUint(uint64(v), 10) }
	return AmneziaWG{PrivateKey: priv, PublicKey: pub, Params: map[string]string{
		"jc": u(randUint(4, 10)), "jmin": u(jmin), "jmax": u(jmin + randUint(30, 120)),
		"s1": u(s1), "s2": u(s2), "h1": u(h[0]), "h2": u(h[1]), "h3": u(h[2]), "h4": u(h[3]),
	}}, nil
}
