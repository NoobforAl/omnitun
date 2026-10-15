// Package classicvpn runs "classic" VPN protocols — SSTP and IKEv2/IPsec —
// in the core, on the pure-Go veepin implementation (third_party/veepin,
// patched so the app supplies the TUN device). Unlike the OS's IKEv2 client
// it works on every Android phone, including ones without IPsec support.
package classicvpn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xen0bit/veepin/client"
	"github.com/xen0bit/veepin/dataplane"
	"github.com/xen0bit/veepin/ikev2"
	"github.com/xen0bit/veepin/sstp"

	"github.com/NoobforAl/omnitun/core/profile"
)

// Device is a TUN-like packet device (on Android, the VpnService fd).
type Device = dataplane.PacketDevice

// Interface is what a protocol negotiated; the opener creates the device for it.
type Interface = dataplane.TUNRequest

// Opener creates the device once the tunnel's addresses are known.
type Opener func(Interface) (Device, error)

var (
	mu           sync.Mutex
	clientOpener Opener
	// ServerOpener supplies devices for in-process veepin servers (tests).
	ServerOpener Opener
)

func init() {
	dataplane.TUNOpener = func(req dataplane.TUNRequest) (dataplane.PacketDevice, error) {
		mu.Lock()
		open := clientOpener
		if req.Server {
			open = ServerOpener
		}
		mu.Unlock()
		if open == nil {
			return nil, errors.New("no TUN device available")
		}
		return open(req)
	}
}

// Supported reports whether this package runs proto.
func Supported(proto string) bool { return proto == profile.SSTP || proto == profile.IKEv2 }

// Tunnel is a running SSTP or IKEv2 session.
type Tunnel struct {
	session client.Session
	Result  client.Result
	rx, tx  atomic.Int64
	tmp     string
}

// Dial connects p and creates its device with open. Only one Dial may run at a time.
func Dial(ctx context.Context, p *profile.Profile, open Opener, logf func(string, ...any)) (*Tunnel, error) {
	t := &Tunnel{}
	mu.Lock()
	clientOpener = func(req Interface) (Device, error) {
		dev, err := open(req)
		if err != nil {
			return nil, err
		}
		return &counting{Device: dev, t: t, logf: logf}, nil
	}
	mu.Unlock()
	defer func() {
		mu.Lock()
		clientOpener = nil
		mu.Unlock()
	}()
	logger := slog.New(logHandler{logf: logf})

	var err error
	switch p.Protocol {
	case profile.SSTP:
		t.session, t.Result, err = sstp.Dial(ctx, sstp.Config{
			Server: p.Server, Port: p.Port, Username: p.Username, Password: p.Password,
			SkipVerify: p.TLS != nil && p.TLS.Insecure, Logger: logger,
		})
	case profile.IKEv2:
		var cfg ikev2.Config
		if cfg, err = t.ikev2Config(p); err == nil {
			cfg.Logger = logger
			t.session, t.Result, err = ikev2.Dial(ctx, cfg)
		}
	default:
		err = fmt.Errorf("classicvpn: unsupported protocol %q", p.Protocol)
	}
	if err != nil {
		t.cleanup()
		if errors.Is(err, client.ErrAuth) {
			return nil, fmt.Errorf("the server rejected the credentials: %w", err)
		}
		return nil, err
	}
	return t, nil
}

func (t *Tunnel) ikev2Config(p *profile.Profile) (ikev2.Config, error) {
	o := p.IKEv2
	if o == nil {
		return ikev2.Config{}, errors.New("missing IKEv2 settings")
	}
	cfg := ikev2.Config{Server: p.Server, Port: p.Port, ServerID: o.RemoteID, TCP: o.TCP}
	if cfg.Port == 500 && !o.TCP {
		cfg.Port = 0 // default
	}
	cfg.LocalID = o.LocalID
	switch o.Auth {
	case "psk":
		cfg.PSK = o.PSK
	case "eap":
		cfg.PSK = o.PSK // optional: authenticates the server instead of its certificate
		cfg.EAPUser, cfg.EAPPassword = p.Username, p.Password
		if cfg.EAPUser == "" || cfg.EAPPassword == "" {
			return cfg, profile.ErrNeedsCredentials
		}
	case "cert":
	default:
		return cfg, fmt.Errorf("unknown IKEv2 auth %q", o.Auth)
	}
	if cfg.LocalID == "" {
		cfg.LocalID = p.Username
	}
	if cfg.LocalID == "" {
		cfg.LocalID = "omnitun"
	}
	// veepin reads certificates from files.
	write := func(name, pem string) (string, error) {
		if pem == "" {
			return "", nil
		}
		if t.tmp == "" {
			dir, err := os.MkdirTemp("", "omnitun-ikev2-")
			if err != nil {
				return "", err
			}
			t.tmp = dir
		}
		path := filepath.Join(t.tmp, name)
		return path, os.WriteFile(path, []byte(pem), 0o600)
	}
	var err error
	if cfg.CAFile, err = write("ca.pem", o.CACert); err != nil {
		return cfg, err
	}
	if o.Auth == "cert" {
		if cfg.CertFile, err = write("cert.pem", o.ClientCert); err != nil {
			return cfg, err
		}
		if cfg.KeyFile, err = write("key.pem", o.ClientKey); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// Wait blocks until the tunnel ends and returns why.
func (t *Tunnel) Wait(ctx context.Context) error { return t.session.Wait(ctx) }

// Close ends the tunnel.
func (t *Tunnel) Close() error {
	err := t.session.Close()
	t.cleanup()
	return err
}

// Traffic returns bytes received and sent through the tunnel.
func (t *Tunnel) Traffic() (rx, tx int64) { return t.rx.Load(), t.tx.Load() }

func (t *Tunnel) cleanup() {
	if t.tmp != "" {
		os.RemoveAll(t.tmp)
		t.tmp = ""
	}
}

// counting tracks traffic: reads are packets the apps send (tx), writes are
// packets from the server (rx).
type counting struct {
	Device
	t      *Tunnel
	logf   func(string, ...any)
	logged atomic.Bool
}

func (c *counting) Read(b []byte) (int, error) {
	n, err := c.Device.Read(b)
	c.t.tx.Add(int64(n))
	c.logErr("read", err)
	return n, err
}

func (c *counting) Write(b []byte) (int, error) {
	n, err := c.Device.Write(b)
	c.t.rx.Add(int64(n))
	c.logErr("write", err)
	return n, err
}

// logErr reports the first device error (later ones are usually the same).
func (c *counting) logErr(op string, err error) {
	if err != nil && c.logf != nil && !c.logged.Swap(true) {
		c.logf("TUN %s: %v", op, err)
	}
}

// logHandler forwards veepin's warnings and errors to the engine log.
type logHandler struct{ logf func(string, ...any) }

func (h logHandler) Enabled(_ context.Context, l slog.Level) bool {
	return h.logf != nil && l >= slog.LevelWarn
}
func (h logHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
		return true
	})
	h.logf("%s", b.String())
	return nil
}
func (h logHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h logHandler) WithGroup(string) slog.Handler      { return h }

// IPv4Prefix returns the device's IPv4 address as a prefix (host route), or "".
func IPv4Prefix(req Interface) string {
	if ip := req.IPv4.To4(); ip != nil {
		ones := 32
		if m := net.IPMask(req.Netmask.To4()); m != nil && req.Netmask != nil {
			if o, bits := m.Size(); bits == 32 && o > 0 {
				ones = o
			}
		}
		return fmt.Sprintf("%s/%d", ip, ones)
	}
	return ""
}
