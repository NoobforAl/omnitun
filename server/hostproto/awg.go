package hostproto

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"

	"github.com/NoobforAl/omnitun/core/protocols"
)

const (
	awgInterface = "omnitun-awg"
	awgMTU       = 1380
)

// awgParamKeys are the obfuscation parameters, in the order UAPI wants them.
var awgParamKeys = []string{"jc", "jmin", "jmax", "s1", "s2", "h1", "h2", "h3", "h4"}

// openTUN creates the kernel TUN device; tests replace it.
var openTUN = func(name string, mtu int) (tun.Device, error) { return tun.CreateTUN(name, mtu) }

type awgServer struct {
	dev *device.Device
	r   Runner
	// users names the owner of each peer, by the peer's public key in hex.
	users map[string]string
}

// Usage reads each peer's counters from the device. The peers are fixed, so
// the numbers only grow while the server runs.
func (x awgServer) Usage() []Usage {
	raw, err := x.dev.IpcGet()
	if err != nil {
		return nil
	}
	var out []Usage
	cur := -1
	for line := range strings.SplitSeq(raw, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "public_key":
			out = append(out, Usage{User: x.users[v]})
			cur = len(out) - 1
		case "rx_bytes":
			if cur >= 0 {
				out[cur].Up, _ = strconv.ParseUint(v, 10, 64)
			}
		case "tx_bytes":
			if cur >= 0 {
				out[cur].Down, _ = strconv.ParseUint(v, 10, 64)
			}
		}
	}
	kept := out[:0]
	for _, u := range out {
		if u.User != "" {
			kept = append(kept, u)
		}
	}
	return kept
}

func (x awgServer) Close() error {
	x.dev.Close() // also closes the TUN device, which removes the interface
	return nil
}

func (awgServer) Pool() string { return protocols.AWGPool }

func hexKey(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(raw) != 32 {
		return "", fmt.Errorf("invalid key")
	}
	return hex.EncodeToString(raw), nil
}

// awgUAPI is the device configuration: the server key, the obfuscation
// parameters and one peer per user (their WireGuard key, an address in the
// AmneziaWG network).
func awgUAPI(cfg Config) (string, map[string]string, error) {
	a := cfg.Env.AWG
	priv, err := hexKey(a.PrivateKey)
	if err != nil {
		return "", nil, fmt.Errorf("amneziawg: server private key: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\nlisten_port=%d\n", priv, cfg.Port)
	for _, k := range awgParamKeys {
		v, ok := a.Params[k]
		if !ok {
			return "", nil, fmt.Errorf("amneziawg: parameter %s is missing", k)
		}
		if _, err := strconv.ParseUint(v, 10, 32); err != nil {
			return "", nil, fmt.Errorf("amneziawg: invalid %s=%q", k, v)
		}
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	b.WriteString("replace_peers=true\n")
	peers := 0
	owners := map[string]string{}
	for _, u := range cfg.Users {
		if u.WGPublicKey == "" || u.WGAddress == "" {
			continue
		}
		pub, err := hexKey(u.WGPublicKey)
		if err != nil {
			return "", nil, fmt.Errorf("amneziawg: user %s: public key: %w", u.Name, err)
		}
		addr, err := protocols.AWGAddress(u.WGAddress)
		if err != nil {
			return "", nil, fmt.Errorf("amneziawg: user %s: %w", u.Name, err)
		}
		fmt.Fprintf(&b, "public_key=%s\nreplace_allowed_ips=true\nallowed_ip=%s\n", pub, addr)
		owners[pub] = u.Name
		peers++
	}
	if peers == 0 {
		return "", nil, fmt.Errorf("amneziawg: no user has a WireGuard key")
	}
	return b.String(), owners, nil
}

func startAWG(cfg Config) (Server, error) {
	uapi, owners, err := awgUAPI(cfg)
	if err != nil {
		return nil, err
	}
	r := cfg.Runner
	if r == nil {
		r = ExecRunner{}
	}
	t, err := openTUN(awgInterface, awgMTU)
	if err != nil {
		return nil, fmt.Errorf("amneziawg: open TUN device: %w", err)
	}
	logger := &device.Logger{Verbosef: func(string, ...any) {}, Errorf: func(f string, args ...any) {
		if cfg.Logf != nil {
			cfg.Logf("amneziawg: "+f, args...)
		}
	}}
	dev := device.NewDevice(t, conn.NewDefaultBind(), logger)
	fail := func(err error) (Server, error) { dev.Close(); return nil, err }
	if err := dev.IpcSet(uapi); err != nil {
		return fail(fmt.Errorf("amneziawg: configure: %w", err))
	}
	if err := dev.Up(); err != nil {
		return fail(fmt.Errorf("amneziawg: bring up: %w", err))
	}
	name := awgInterface
	if n, err := t.Name(); err == nil && n != "" {
		name = n
	}
	// The kernel interface needs the server's address and to be up.
	if err := r.Run("ip", "addr", "add", protocols.AWGServerAddress(), "dev", name); err != nil {
		return fail(fmt.Errorf("amneziawg: %w", err))
	}
	if err := r.Run("ip", "link", "set", "dev", name, "up"); err != nil {
		return fail(fmt.Errorf("amneziawg: %w", err))
	}
	return awgServer{dev: dev, r: r, users: owners}, nil
}
