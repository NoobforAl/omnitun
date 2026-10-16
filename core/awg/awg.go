// Package awg runs AmneziaWG (WireGuard with DPI-resistant junk/obfuscation
// parameters) on a TUN file descriptor provided by the OS (Android VpnService).
// sing-box has no AmneziaWG support, so this is a separate engine.
package awg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"

	"github.com/NoobforAl/omnitun/core/profile"
)

// deviceKeys are the AmneziaWG 1.x device parameters amneziawg-go accepts, in UAPI order.
var deviceKeys = []string{"jc", "jmin", "jmax", "s1", "s2", "h1", "h2", "h3", "h4", "itime"}

// chainKeys are AmneziaWG 1.5's signature packets (i1-i5, sent before the
// handshake to look like another protocol) and controlled junk (j1-j3). Their
// values are tag chains such as "<b 0xc0ff><r 16><t>", passed on verbatim.
var chainKeys = []string{"i1", "i2", "i3", "i4", "i5", "j1", "j2", "j3"}

// ErrUnsupported is returned for parameters this AmneziaWG version can't apply.
var ErrUnsupported = errors.New("unsupported AmneziaWG parameter")

func keyHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(raw) != 32 {
		return "", fmt.Errorf("invalid key %q", b64)
	}
	return hex.EncodeToString(raw), nil
}

// UAPI builds the device configuration in WireGuard's UAPI text format.
func UAPI(p *profile.Profile, endpoint netip.AddrPort) (string, error) {
	wg := p.WireGuard
	if wg == nil {
		return "", errors.New("not a WireGuard/AmneziaWG profile")
	}
	priv, err := keyHex(wg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("private key: %w", err)
	}
	pub, err := keyHex(wg.PeerPublicKey)
	if err != nil {
		return "", fmt.Errorf("peer public key: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\n", priv)
	for _, k := range deviceKeys {
		if v, ok := wg.Amnezia[k]; ok {
			if _, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32); err != nil {
				// AmneziaWG 2.0 writes headers as ranges ("h1=100-200").
				if strings.Contains(v, "-") {
					return "", fmt.Errorf("%w %s=%q (AmneziaWG 2.0 header ranges)", ErrUnsupported, k, v)
				}
				return "", fmt.Errorf("invalid %s=%q", k, v)
			}
			fmt.Fprintf(&b, "%s=%s\n", k, strings.TrimSpace(v))
		}
	}
	for _, k := range chainKeys {
		if v, ok := wg.Amnezia[k]; ok && strings.TrimSpace(v) != "" {
			if strings.ContainsAny(v, "\r\n") {
				return "", fmt.Errorf("invalid %s: line break", k)
			}
			fmt.Fprintf(&b, "%s=%s\n", k, strings.TrimSpace(v))
		}
	}
	for k := range wg.Amnezia {
		if !isDeviceKey(k) && !slices.Contains(chainKeys, k) {
			return "", fmt.Errorf("%w %q (AmneziaWG 2.0 options are not supported yet)", ErrUnsupported, k)
		}
	}
	b.WriteString("replace_peers=true\n")
	fmt.Fprintf(&b, "public_key=%s\n", pub)
	if wg.PreSharedKey != "" {
		psk, err := keyHex(wg.PreSharedKey)
		if err != nil {
			return "", fmt.Errorf("preshared key: %w", err)
		}
		fmt.Fprintf(&b, "preshared_key=%s\n", psk)
	}
	fmt.Fprintf(&b, "endpoint=%s\n", endpoint)
	keepalive := wg.Keepalive
	if keepalive == 0 {
		keepalive = 25 // keeps NAT mappings alive on mobile networks
	}
	fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", keepalive)
	b.WriteString("replace_allowed_ips=true\n")
	allowed := wg.AllowedIPs
	if len(allowed) == 0 {
		allowed = []string{"0.0.0.0/0", "::/0"}
	}
	for _, a := range allowed {
		fmt.Fprintf(&b, "allowed_ip=%s\n", strings.TrimSpace(a))
	}
	return b.String(), nil
}

func isDeviceKey(k string) bool { return slices.Contains(deviceKeys, k) }

// ResolveEndpoint returns the peer address; host names are resolved with the
// system resolver (before the tunnel exists).
func ResolveEndpoint(ctx context.Context, p *profile.Profile) (netip.AddrPort, error) {
	if addr, err := netip.ParseAddr(p.Server); err == nil {
		return netip.AddrPortFrom(addr.Unmap(), uint16(p.Port)), nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", p.Server)
	if err != nil || len(addrs) == 0 {
		return netip.AddrPort{}, fmt.Errorf("resolve %s: %w", p.Server, err)
	}
	// Prefer IPv4: many mobile networks have broken IPv6.
	for _, a := range addrs {
		if a.Unmap().Is4() {
			return netip.AddrPortFrom(a.Unmap(), uint16(p.Port)), nil
		}
	}
	return netip.AddrPortFrom(addrs[0], uint16(p.Port)), nil
}

// Tunnel is a running AmneziaWG device.
type Tunnel struct {
	dev *device.Device
}

// Start creates the device on tunFD (which it duplicates; the caller keeps
// its own descriptor), applies the profile and brings it up. protect is
// called with the UDP socket descriptors so they bypass the VPN.
func Start(p *profile.Profile, endpoint netip.AddrPort, tunFD int, protect func(fd int) error, logf func(string, ...any)) (*Tunnel, error) {
	uapi, err := UAPI(p, endpoint)
	if err != nil {
		return nil, err
	}
	tunDev, err := createTUN(tunFD)
	if err != nil {
		return nil, fmt.Errorf("open tun: %w", err)
	}
	logger := &device.Logger{
		Verbosef: func(string, ...any) {},
		Errorf: func(format string, args ...any) {
			if logf != nil {
				logf("awg: "+format, args...)
			}
		},
	}
	bind := conn.NewDefaultBind()
	dev := device.NewDevice(tunDev, bind, logger)
	if err := dev.IpcSet(uapi); err != nil {
		dev.Close()
		return nil, fmt.Errorf("configure: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("bring up: %w", err)
	}
	if err := protectBind(bind, protect); err != nil {
		dev.Close()
		return nil, fmt.Errorf("protect socket: %w", err)
	}
	return &Tunnel{dev: dev}, nil
}

// Stats is a snapshot of the peer.
type Stats struct {
	RxBytes, TxBytes int64
	LastHandshake    time.Time
}

// Stats reads counters from the device.
func (t *Tunnel) Stats() Stats {
	raw, err := t.dev.IpcGet()
	if err != nil {
		return Stats{}
	}
	return parseStats(raw)
}

func parseStats(raw string) Stats {
	var s Stats
	var sec, nsec int64
	for line := range strings.SplitSeq(raw, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		n, _ := strconv.ParseInt(v, 10, 64)
		switch k {
		case "rx_bytes":
			s.RxBytes += n
		case "tx_bytes":
			s.TxBytes += n
		case "last_handshake_time_sec":
			sec = n
		case "last_handshake_time_nsec":
			nsec = n
		}
	}
	if sec > 0 {
		s.LastHandshake = time.Unix(sec, nsec)
	}
	return s
}

// Close shuts the device down and closes its TUN.
func (t *Tunnel) Close() { t.dev.Close() }
