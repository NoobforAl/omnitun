package profile

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Kinds of form fields.
const (
	FieldText      = "text"
	FieldSecret    = "secret"    // hidden input (passwords, keys)
	FieldInt       = "int"       // whole number
	FieldBool      = "bool"      // "true" / "false"
	FieldChoice    = "choice"    // one of Choices ("" allowed unless Required)
	FieldMultiline = "multiline" // PEM blocks, bridge lines
	FieldList      = "list"      // comma-separated values
)

// Field describes one input of the profile editor. Forms are generated from
// this table, so every app and CLI edits profiles the same way.
type Field struct {
	// Key is a stable name ("uuid", "tls.server_name", "amnezia.jc").
	Key   string `json:"key"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
	Kind  string `json:"kind" enum:"text,secret,int,bool,choice,multiline,list"`
	// Required fields must be filled; Profile.Validate checks the rest
	// (for example SSH needs a password or a key).
	Required bool     `json:"required,omitempty"`
	Choices  []string `json:"choices,omitempty"`
	Default  string   `json:"default,omitempty"`
	// Advanced fields can be hidden behind a "more" toggle.
	Advanced bool `json:"advanced,omitempty"`

	get func(*Profile) string
	set func(*Profile, string) error
}

// ProtocolForm is the editor of one protocol.
type ProtocolForm struct {
	Protocol string  `json:"protocol"`
	Label    string  `json:"label"`
	Fields   []Field `json:"fields"`
}

// ------------------------------------------------------------- accessors

type strPtr func(p *Profile, create bool) *string
type intPtr func(p *Profile, create bool) *int
type boolPtr func(p *Profile, create bool) *bool
type listPtr func(p *Profile, create bool) *[]string

func (f Field) with(opts ...func(*Field)) Field {
	for _, o := range opts {
		o(&f)
	}
	return f
}

func required(f *Field)          { f.Required = true }
func advanced(f *Field)          { f.Advanced = true }
func help(s string) func(*Field) { return func(f *Field) { f.Help = s } }
func def(s string) func(*Field)  { return func(f *Field) { f.Default = s } }
func kind(k string) func(*Field) { return func(f *Field) { f.Kind = k } }
func choices(c ...string) func(*Field) {
	return func(f *Field) { f.Kind, f.Choices = FieldChoice, c }
}

func text(key, label string, ptr strPtr, opts ...func(*Field)) Field {
	return Field{
		Key: key, Label: label, Kind: FieldText,
		get: func(p *Profile) string {
			if s := ptr(p, false); s != nil {
				return *s
			}
			return ""
		},
		set: func(p *Profile, v string) error {
			v = strings.TrimSpace(v)
			if v == "" {
				if s := ptr(p, false); s != nil {
					*s = ""
				}
				return nil
			}
			*ptr(p, true) = v
			return nil
		},
	}.with(opts...)
}

// pem is a multi-line text field: surrounding space is trimmed but line
// breaks inside are kept.
func pem(key, label string, ptr strPtr, opts ...func(*Field)) Field {
	f := text(key, label, ptr, kind(FieldMultiline))
	return f.with(opts...)
}

func secret(key, label string, ptr strPtr, opts ...func(*Field)) Field {
	return text(key, label, ptr, append([]func(*Field){kind(FieldSecret)}, opts...)...)
}

func number(key, label string, ptr intPtr, opts ...func(*Field)) Field {
	return Field{
		Key: key, Label: label, Kind: FieldInt,
		get: func(p *Profile) string {
			if n := ptr(p, false); n != nil && *n != 0 {
				return strconv.Itoa(*n)
			}
			return ""
		},
		set: func(p *Profile, v string) error {
			v = strings.TrimSpace(v)
			if v == "" {
				if n := ptr(p, false); n != nil {
					*n = 0
				}
				return nil
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return formErr(FormNotNumber, "%s must be a whole number", label).arg("label", label)
			}
			*ptr(p, true) = n
			return nil
		},
	}.with(opts...)
}

func toggle(key, label string, ptr boolPtr, opts ...func(*Field)) Field {
	return Field{
		Key: key, Label: label, Kind: FieldBool,
		get: func(p *Profile) string {
			if b := ptr(p, false); b != nil && *b {
				return "true"
			}
			return "false"
		},
		set: func(p *Profile, v string) error {
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "1", "yes", "on":
				*ptr(p, true) = true
			case "", "false", "0", "no", "off":
				if b := ptr(p, false); b != nil {
					*b = false
				}
			default:
				return formErr(FormNotBool, "%s must be true or false", label).arg("label", label)
			}
			return nil
		},
	}.with(opts...)
}

// splitList splits on commas and line breaks, dropping empty items.
func splitList(v string) []string {
	var out []string
	for _, item := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func list(key, label string, ptr listPtr, sep string, opts ...func(*Field)) Field {
	return Field{
		Key: key, Label: label, Kind: FieldList,
		get: func(p *Profile) string {
			if l := ptr(p, false); l != nil {
				return strings.Join(*l, sep)
			}
			return ""
		},
		set: func(p *Profile, v string) error {
			items := splitList(v)
			if len(items) == 0 {
				if l := ptr(p, false); l != nil {
					*l = nil
				}
				return nil
			}
			*ptr(p, true) = items
			return nil
		},
	}.with(opts...)
}

// -------------------------------------------------- nested structure access

func tlsOf(p *Profile, create bool) *TLS {
	if p.TLS == nil && create {
		p.TLS = &TLS{Enabled: true}
	}
	return p.TLS
}

func tlsStr(sel func(*TLS) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if t := tlsOf(p, create); t != nil {
			return sel(t)
		}
		return nil
	}
}

func realityStr(sel func(*Reality) *string) strPtr {
	return func(p *Profile, create bool) *string {
		t := tlsOf(p, create)
		if t == nil {
			return nil
		}
		if t.Reality == nil {
			if !create {
				return nil
			}
			t.Reality = &Reality{}
		}
		return sel(t.Reality)
	}
}

func transportStr(sel func(*Transport) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if p.Transport == nil {
			if !create {
				return nil
			}
			p.Transport = &Transport{}
		}
		return sel(p.Transport)
	}
}

func wgOf(p *Profile, create bool) *WG {
	if p.WireGuard == nil && create {
		p.WireGuard = &WG{}
	}
	return p.WireGuard
}

func wgStr(sel func(*WG) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if w := wgOf(p, create); w != nil {
			return sel(w)
		}
		return nil
	}
}

func wgList(sel func(*WG) *[]string) listPtr {
	return func(p *Profile, create bool) *[]string {
		if w := wgOf(p, create); w != nil {
			return sel(w)
		}
		return nil
	}
}

func hy2Of(p *Profile, create bool) *Hy2 {
	if p.Hysteria2 == nil && create {
		p.Hysteria2 = &Hy2{}
	}
	return p.Hysteria2
}

func hy2Str(sel func(*Hy2) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if h := hy2Of(p, create); h != nil {
			return sel(h)
		}
		return nil
	}
}

func hy2Int(sel func(*Hy2) *int) intPtr {
	return func(p *Profile, create bool) *int {
		if h := hy2Of(p, create); h != nil {
			return sel(h)
		}
		return nil
	}
}

func tuicStr(sel func(*TUICOpts) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if p.TUIC == nil {
			if !create {
				return nil
			}
			p.TUIC = &TUICOpts{}
		}
		return sel(p.TUIC)
	}
}

func ovpnOf(p *Profile, create bool) *OVPN {
	if p.OpenVPN == nil && create {
		p.OpenVPN = &OVPN{Network: "udp"}
	}
	return p.OpenVPN
}

func ovpnStr(sel func(*OVPN) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if o := ovpnOf(p, create); o != nil {
			return sel(o)
		}
		return nil
	}
}

func ikeOf(p *Profile, create bool) *IKEv2Opts {
	if p.IKEv2 == nil && create {
		p.IKEv2 = &IKEv2Opts{Auth: "eap"}
	}
	return p.IKEv2
}

func ikeStr(sel func(*IKEv2Opts) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if o := ikeOf(p, create); o != nil {
			return sel(o)
		}
		return nil
	}
}

func sshOf(p *Profile, create bool) *SSHOpts {
	if p.SSH == nil && create {
		p.SSH = &SSHOpts{}
	}
	return p.SSH
}

func sshStr(sel func(*SSHOpts) *string) strPtr {
	return func(p *Profile, create bool) *string {
		if o := sshOf(p, create); o != nil {
			return sel(o)
		}
		return nil
	}
}

func torOf(p *Profile, create bool) *TorOpts {
	if p.Tor == nil && create {
		p.Tor = &TorOpts{}
	}
	return p.Tor
}

// amnezia is an AmneziaWG parameter stored in a map.
func amnezia(key, label string, opts ...func(*Field)) Field {
	k := strings.TrimPrefix(key, "amnezia.")
	return Field{
		Key: key, Label: label, Kind: FieldText, Advanced: true,
		get: func(p *Profile) string {
			if p.WireGuard == nil {
				return ""
			}
			return p.WireGuard.Amnezia[k]
		},
		set: func(p *Profile, v string) error {
			v = strings.TrimSpace(v)
			if v == "" {
				if p.WireGuard != nil {
					delete(p.WireGuard.Amnezia, k)
				}
				return nil
			}
			w := wgOf(p, true)
			if w.Amnezia == nil {
				w.Amnezia = map[string]string{}
			}
			w.Amnezia[k] = v
			return nil
		},
	}.with(opts...)
}

// security is the transport security of TLS-capable protocols: none, tls or
// (VLESS) reality.
func security(withReality bool) Field {
	ch := []string{"none", "tls"}
	if withReality {
		ch = append(ch, "reality")
	}
	return Field{
		Key: "security", Label: "Security", Kind: FieldChoice, Choices: ch, Default: "tls",
		get: func(p *Profile) string {
			switch {
			case p.TLS == nil || !p.TLS.Enabled:
				return "none"
			case p.TLS.Reality != nil:
				return "reality"
			}
			return "tls"
		},
		set: func(p *Profile, v string) error {
			switch strings.TrimSpace(v) {
			case "none":
				p.TLS = nil
			case "tls", "":
				t := tlsOf(p, true)
				t.Enabled, t.Reality = true, nil
			case "reality":
				if !withReality {
					return formErr(FormRealityOnlyVLESS, "reality is only available for VLESS")
				}
				t := tlsOf(p, true)
				t.Enabled = true
				if t.Reality == nil {
					t.Reality = &Reality{}
				}
			default:
				return formErr(FormUnknownSecurity, "unknown security %q", v).arg("value", v)
			}
			return nil
		},
	}
}

// reserved is WireGuard's reserved bytes (Cloudflare WARP), three numbers.
func reserved() Field {
	return Field{
		Key: "reserved", Label: "Reserved bytes", Kind: FieldList, Advanced: true,
		Help: "Three numbers, as WARP configurations list them",
		get: func(p *Profile) string {
			if p.WireGuard == nil || len(p.WireGuard.Reserved) == 0 {
				return ""
			}
			parts := make([]string, len(p.WireGuard.Reserved))
			for i, n := range p.WireGuard.Reserved {
				parts[i] = strconv.Itoa(n)
			}
			return strings.Join(parts, ", ")
		},
		set: func(p *Profile, v string) error {
			items := splitList(v)
			if len(items) == 0 {
				if p.WireGuard != nil {
					p.WireGuard.Reserved = nil
				}
				return nil
			}
			var out []int
			for _, it := range items {
				n, err := strconv.Atoi(it)
				if err != nil || n < 0 || n > 255 {
					return formErr(FormReservedBytes, "reserved bytes must be numbers from 0 to 255")
				}
				out = append(out, n)
			}
			wgOf(p, true).Reserved = out
			return nil
		},
	}
}

// ------------------------------------------------------- the field tables

var fingerprints = []string{"", "chrome", "firefox", "safari", "ios", "android", "edge", "random", "randomized"}

func tlsFields(withSecurity, withReality bool) []Field {
	var fs []Field
	if withSecurity {
		fs = append(fs, security(withReality))
	}
	fs = append(fs,
		text("tls.server_name", "TLS server name (SNI)", tlsStr(func(t *TLS) *string { return &t.ServerName }), advanced,
			help("Defaults to the server address")),
		text("tls.fingerprint", "TLS fingerprint", tlsStr(func(t *TLS) *string { return &t.Fingerprint }), advanced,
			choices(fingerprints...), help("Which browser's TLS handshake to imitate")),
		list("tls.alpn", "ALPN", func(p *Profile, c bool) *[]string {
			if t := tlsOf(p, c); t != nil {
				return &t.ALPN
			}
			return nil
		}, ", ", advanced),
		toggle("tls.insecure", "Skip certificate check", func(p *Profile, c bool) *bool {
			if t := tlsOf(p, c); t != nil {
				return &t.Insecure
			}
			return nil
		}, advanced, help("Only for servers with a self-signed certificate you cannot pin")),
		pem("tls.certificate", "Pinned certificate (PEM)", tlsStr(func(t *TLS) *string { return &t.Certificate }), advanced,
			help("Trust this certificate instead of the system's authorities")),
	)
	if withReality {
		fs = append(fs,
			text("reality.public_key", "Reality public key", realityStr(func(r *Reality) *string { return &r.PublicKey }),
				help("Needed when Security is reality")),
			text("reality.short_id", "Reality short id", realityStr(func(r *Reality) *string { return &r.ShortID })),
		)
	}
	return fs
}

func transportFields() []Field {
	return []Field{
		text("transport.type", "Transport", transportStr(func(t *Transport) *string { return &t.Type }), advanced,
			choices("", "ws", "grpc", "httpupgrade", "http")),
		text("transport.path", "Path", transportStr(func(t *Transport) *string { return &t.Path }), advanced),
		text("transport.host", "Host header", transportStr(func(t *Transport) *string { return &t.Host }), advanced),
		text("transport.service_name", "gRPC service name", transportStr(func(t *Transport) *string { return &t.ServiceName }), advanced),
	}
}

func password(opts ...func(*Field)) Field {
	return secret("password", "Password", func(p *Profile, _ bool) *string { return &p.Password }, opts...)
}

func username(opts ...func(*Field)) Field {
	return text("username", "Username", func(p *Profile, _ bool) *string { return &p.Username }, opts...)
}

func uuid() Field {
	return text("uuid", "UUID", func(p *Profile, _ bool) *string { return &p.UUID }, required)
}

var ssMethods = []string{
	"2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305",
	"aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305", "none",
}

func wireguardFields(amneziaToo bool) []Field {
	fs := []Field{
		secret("private_key", "Private key", wgStr(func(w *WG) *string { return &w.PrivateKey }), required),
		text("peer_public_key", "Server public key", wgStr(func(w *WG) *string { return &w.PeerPublicKey }), required),
		secret("pre_shared_key", "Pre-shared key", wgStr(func(w *WG) *string { return &w.PreSharedKey }), advanced),
		list("address", "Your tunnel address", wgList(func(w *WG) *[]string { return &w.Address }), ", ", required,
			help("For example 10.8.0.2/32")),
		list("allowed_ips", "Routed networks", wgList(func(w *WG) *[]string { return &w.AllowedIPs }), ", ", advanced,
			def("0.0.0.0/0, ::/0")),
		list("dns", "DNS servers", wgList(func(w *WG) *[]string { return &w.DNS }), ", ", advanced),
		number("mtu", "MTU", func(p *Profile, c bool) *int {
			if w := wgOf(p, c); w != nil {
				return &w.MTU
			}
			return nil
		}, advanced),
		number("keepalive", "Keepalive (seconds)", func(p *Profile, c bool) *int {
			if w := wgOf(p, c); w != nil {
				return &w.Keepalive
			}
			return nil
		}, advanced),
		reserved(),
	}
	if amneziaToo {
		fs = append(fs,
			amnezia("amnezia.jc", "Jc (junk packet count)"), amnezia("amnezia.jmin", "Jmin"), amnezia("amnezia.jmax", "Jmax"),
			amnezia("amnezia.s1", "S1"), amnezia("amnezia.s2", "S2"),
			amnezia("amnezia.h1", "H1"), amnezia("amnezia.h2", "H2"), amnezia("amnezia.h3", "H3"), amnezia("amnezia.h4", "H4"),
			amnezia("amnezia.i1", "I1 (signature packet)"), amnezia("amnezia.i2", "I2"), amnezia("amnezia.i3", "I3"),
			amnezia("amnezia.i4", "I4"), amnezia("amnezia.i5", "I5"),
			amnezia("amnezia.j1", "J1 (controlled junk)"), amnezia("amnezia.j2", "J2"), amnezia("amnezia.j3", "J3"),
			amnezia("amnezia.itime", "Itime"),
		)
	}
	return fs
}

// protocolFields returns the fields of one protocol, after the common
// name/server/port.
func protocolFields(protocol string) ([]Field, bool) {
	switch protocol {
	case VLESS:
		fs := []Field{uuid(),
			text("flow", "Flow", func(p *Profile, _ bool) *string { return &p.Flow }, choices("", "xtls-rprx-vision"), advanced)}
		fs = append(fs, tlsFields(true, true)...)
		return append(fs, transportFields()...), true
	case VMess:
		fs := []Field{uuid(),
			text("method", "Encryption", func(p *Profile, _ bool) *string { return &p.Method }, choices("auto", "aes-128-gcm", "chacha20-poly1305", "none", "zero"), def("auto")),
			number("alter_id", "Alter id", func(p *Profile, _ bool) *int { return &p.AlterID }, advanced)}
		fs = append(fs, tlsFields(true, false)...)
		return append(fs, transportFields()...), true
	case Trojan:
		fs := []Field{password(required)}
		fs = append(fs, tlsFields(true, false)...)
		return append(fs, transportFields()...), true
	case Shadowsocks:
		return []Field{
			text("method", "Cipher", func(p *Profile, _ bool) *string { return &p.Method }, required, choices(ssMethods...), def("2022-blake3-aes-128-gcm")),
			password(required),
			text("plugin", "Plugin", func(p *Profile, _ bool) *string { return &p.Plugin }, advanced, help("For example shadow-tls or v2ray-plugin")),
			text("plugin_opts", "Plugin options", func(p *Profile, _ bool) *string { return &p.PluginOpts }, advanced),
		}, true
	case Hysteria2, Hysteria:
		fs := []Field{password(required),
			text("hysteria.obfs", "Obfuscation", hy2Str(func(h *Hy2) *string { return &h.Obfs }), advanced,
				help("salamander for Hysteria 2; the obfuscation password for Hysteria 1")),
			secret("hysteria.obfs_password", "Obfuscation password", hy2Str(func(h *Hy2) *string { return &h.ObfsPassword }), advanced),
			number("hysteria.up_mbps", "Upload (Mbps)", hy2Int(func(h *Hy2) *int { return &h.UpMbps }), advanced),
			number("hysteria.down_mbps", "Download (Mbps)", hy2Int(func(h *Hy2) *int { return &h.DownMbps }), advanced),
		}
		return append(fs, tlsFields(false, false)...), true
	case TUIC:
		fs := []Field{uuid(), password(),
			text("tuic.congestion_control", "Congestion control", tuicStr(func(t *TUICOpts) *string { return &t.CongestionControl }),
				choices("", "bbr", "cubic", "new_reno"), advanced),
			text("tuic.udp_relay_mode", "UDP relay mode", tuicStr(func(t *TUICOpts) *string { return &t.UDPRelayMode }),
				choices("", "native", "quic"), advanced),
		}
		return append(fs, tlsFields(false, false)...), true
	case AnyTLS:
		return append([]Field{password(required)}, tlsFields(false, false)...), true
	case Naive:
		fs := []Field{username(required), password(required),
			toggle("naive.quic", "Use HTTP/3 (QUIC)", func(p *Profile, c bool) *bool {
				if p.Naive == nil {
					if !c {
						return nil
					}
					p.Naive = &NaiveOpts{}
				}
				return &p.Naive.QUIC
			}, advanced)}
		return append(fs, tlsFields(false, false)...), true
	case SSH:
		return []Field{username(required), password(),
			pem("ssh.private_key", "Private key (PEM)", sshStr(func(o *SSHOpts) *string { return &o.PrivateKey })),
			secret("ssh.private_key_passphrase", "Key passphrase", sshStr(func(o *SSHOpts) *string { return &o.PrivateKeyPassphrase }), advanced),
			list("ssh.host_key", "Host keys", func(p *Profile, c bool) *[]string {
				if o := sshOf(p, c); o != nil {
					return &o.HostKey
				}
				return nil
			}, "\n", advanced, kind(FieldMultiline), help("authorized_keys format; empty accepts any key")),
		}, true
	case WireGuard:
		return wireguardFields(false), true
	case AmneziaWG:
		return wireguardFields(true), true
	case OpenVPN:
		return []Field{
			username(), password(),
			text("openvpn.network", "Transport", ovpnStr(func(o *OVPN) *string { return &o.Network }), choices("udp", "tcp"), def("udp")),
			pem("openvpn.ca", "CA certificate (PEM)", ovpnStr(func(o *OVPN) *string { return &o.CA }), required),
			pem("openvpn.cert", "Client certificate (PEM)", ovpnStr(func(o *OVPN) *string { return &o.Cert })),
			pem("openvpn.key", "Client key (PEM)", ovpnStr(func(o *OVPN) *string { return &o.Key })),
			text("openvpn.control_wrap", "Control channel protection", ovpnStr(func(o *OVPN) *string { return &o.ControlWrap }),
				choices("", "tls-auth", "tls-crypt", "tls-crypt-v2"), advanced),
			pem("openvpn.control_wrap_key", "Control channel key", ovpnStr(func(o *OVPN) *string { return &o.ControlWrapKey }), advanced),
			text("openvpn.key_direction", "Key direction", ovpnStr(func(o *OVPN) *string { return &o.KeyDirection }), choices("", "0", "1"), advanced),
			text("openvpn.cipher", "Cipher", ovpnStr(func(o *OVPN) *string { return &o.Cipher }), advanced),
			text("openvpn.auth", "Auth digest", ovpnStr(func(o *OVPN) *string { return &o.Auth }), advanced),
			text("openvpn.verify_x509_name", "Verify server name", ovpnStr(func(o *OVPN) *string { return &o.VerifyX509Name }), advanced),
			text("openvpn.remote_cert_tls", "Remote certificate type", ovpnStr(func(o *OVPN) *string { return &o.RemoteCertTLS }), choices("", "server"), advanced),
		}, true
	case IKEv2:
		return []Field{
			text("ikev2.auth", "Authentication", ikeStr(func(o *IKEv2Opts) *string { return &o.Auth }), choices("eap", "psk", "cert"), def("eap"), required),
			username(), password(),
			secret("ikev2.psk", "Pre-shared key", ikeStr(func(o *IKEv2Opts) *string { return &o.PSK })),
			text("ikev2.remote_id", "Server identity", ikeStr(func(o *IKEv2Opts) *string { return &o.RemoteID }), advanced, help("Defaults to the server address")),
			text("ikev2.local_id", "Your identity", ikeStr(func(o *IKEv2Opts) *string { return &o.LocalID }), advanced),
			pem("ikev2.ca_cert", "Server certificate to trust (PEM)", ikeStr(func(o *IKEv2Opts) *string { return &o.CACert }), advanced),
			pem("ikev2.client_cert", "Client certificate (PEM)", ikeStr(func(o *IKEv2Opts) *string { return &o.ClientCert }), advanced),
			pem("ikev2.client_key", "Client key (PEM)", ikeStr(func(o *IKEv2Opts) *string { return &o.ClientKey }), advanced),
			toggle("ikev2.tcp", "Use TCP", func(p *Profile, c bool) *bool {
				if o := ikeOf(p, c); o != nil {
					return &o.TCP
				}
				return nil
			}, advanced, help("Carry IKE and ESP over TCP for networks that block UDP")),
			toggle("ikev2.system", "Use the phone's own IKEv2 client", func(p *Profile, c bool) *bool {
				if o := ikeOf(p, c); o != nil {
					return &o.System
				}
				return nil
			}, advanced),
		}, true
	case SSTP:
		return []Field{username(required), password(required),
			toggle("tls.insecure", "Skip certificate check", func(p *Profile, c bool) *bool {
				if t := tlsOf(p, c); t != nil {
					return &t.Insecure
				}
				return nil
			}, advanced, help("Safe for SSTP: the password exchange proves the server")),
		}, true
	case Tor:
		return []Field{
			text("tor.builtin", "Bridges", func(p *Profile, c bool) *string {
				if o := torOf(p, c); o != nil {
					return &o.Builtin
				}
				return nil
			}, choices("", "snowflake", "obfs4"), help("Empty connects to Tor directly, or uses the bridge lines below")),
			list("tor.bridges", "Bridge lines", func(p *Profile, c bool) *[]string {
				if o := torOf(p, c); o != nil {
					return &o.Bridges
				}
				return nil
			}, "\n", kind(FieldMultiline), help("One bridge per line, as in a torrc without \"Bridge \"")),
		}, true
	}
	return nil, false
}

// formLabels name the protocols in the order an app lists them.
var formLabels = []struct{ protocol, label string }{
	{VLESS, "VLESS"}, {VMess, "VMess"}, {Trojan, "Trojan"}, {Shadowsocks, "Shadowsocks"},
	{Hysteria2, "Hysteria 2"}, {Hysteria, "Hysteria"}, {TUIC, "TUIC"}, {AnyTLS, "AnyTLS"}, {Naive, "NaiveProxy"},
	{WireGuard, "WireGuard"}, {AmneziaWG, "AmneziaWG"}, {OpenVPN, "OpenVPN"}, {IKEv2, "IKEv2/IPsec"}, {SSTP, "SSTP"},
	{SSH, "SSH"}, {Tor, "Tor"},
}

// defaultPorts are the ports a new profile starts with.
var defaultPorts = map[string]int{
	VLESS: 443, VMess: 443, Trojan: 443, Shadowsocks: 8388, Hysteria2: 443, Hysteria: 443, TUIC: 443,
	AnyTLS: 443, Naive: 443, SSH: 22, WireGuard: 51820, AmneziaWG: 51820, OpenVPN: 1194, IKEv2: 4500, SSTP: 443,
}

func commonFields(protocol string) []Field {
	fs := []Field{text("name", "Name", func(p *Profile, _ bool) *string { return &p.Name })}
	if protocol == Tor {
		return fs
	}
	return append(fs,
		text("server", "Server", func(p *Profile, _ bool) *string { return &p.Server }, required, help("Host name or IP address")),
		number("port", "Port", func(p *Profile, _ bool) *int { return &p.Port }, required),
	)
}

// FormFor returns the fields of the editor for a protocol, or false when the
// protocol has no form.
func FormFor(protocol string) ([]Field, bool) {
	own, ok := protocolFields(protocol)
	if !ok {
		return nil, false
	}
	fs := append(commonFields(protocol), own...)
	for i := range fs {
		if fs[i].Kind == FieldChoice && fs[i].Default == "" && len(fs[i].Choices) > 0 && fs[i].Choices[0] != "" {
			fs[i].Default = fs[i].Choices[0]
		}
	}
	return fs, true
}

// Forms lists the editor of every protocol, in display order.
func Forms() []ProtocolForm {
	out := make([]ProtocolForm, 0, len(formLabels))
	for _, l := range formLabels {
		fs, _ := FormFor(l.protocol)
		out = append(out, ProtocolForm{Protocol: l.protocol, Label: l.label, Fields: fs})
	}
	return out
}

// New returns a profile of the protocol with its default port and defaults
// (TLS on for the protocols that always use it).
func New(protocol string) (*Profile, error) {
	if _, ok := FormFor(protocol); !ok {
		return nil, fmt.Errorf("there is no editor for %q", protocol)
	}
	p := &Profile{Protocol: protocol, Port: defaultPorts[protocol]}
	switch protocol {
	case Trojan, Hysteria2, Hysteria, TUIC, AnyTLS, Naive:
		p.TLS = &TLS{Enabled: true}
	case VMess:
		p.Method = "auto"
	case IKEv2:
		p.IKEv2 = &IKEv2Opts{Auth: "eap"}
	case OpenVPN:
		p.OpenVPN = &OVPN{Network: "udp"}
	case SSTP:
		p.TLS = &TLS{Enabled: true}
	}
	return p, nil
}

// Values returns the profile's field values by key (every field of its form,
// empty strings included).
func (p *Profile) Values() map[string]string {
	out := map[string]string{}
	fs, _ := FormFor(p.Protocol)
	for _, f := range fs {
		out[f.Key] = f.get(p)
	}
	return out
}

// Apply sets fields from values. Unknown keys are an error; an empty value
// clears the field. Fields are applied in form order, so "security" takes
// effect before the TLS details.
func (p *Profile) Apply(values map[string]string) error {
	fs, ok := FormFor(p.Protocol)
	if !ok {
		return fmt.Errorf("there is no editor for %q", p.Protocol)
	}
	known := map[string]bool{}
	for _, f := range fs {
		known[f.Key] = true
	}
	var unknown []string
	for k := range values {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("%s has no setting %q", p.Protocol, unknown[0])
	}
	for _, f := range fs {
		v, given := values[f.Key]
		if !given {
			continue
		}
		if f.Kind == FieldChoice && v != "" && !slices.Contains(f.Choices, v) {
			return formErr(FormBadChoice, "%s: %q is not one of %s", f.Label, v, strings.Join(f.Choices, ", ")).
				arg("label", f.Label).arg("value", v).arg("choices", strings.Join(f.Choices, ", "))
		}
		if err := f.set(p, v); err != nil {
			return err
		}
	}
	return nil
}

// FromValues builds a profile from an editor's values: defaults, the values,
// the required fields and the protocol's own checks.
func FromValues(protocol string, values map[string]string) (*Profile, error) {
	p, err := New(protocol)
	if err != nil {
		return nil, err
	}
	if err := p.Apply(values); err != nil {
		return nil, err
	}
	fs, _ := FormFor(protocol)
	for _, f := range fs {
		if f.Required && strings.TrimSpace(f.get(p)) == "" {
			return nil, formErr(FormRequired, "%s is required", f.Label).arg("label", f.Label)
		}
	}
	if p.TLS != nil && p.TLS.Reality != nil && p.TLS.Reality.PublicKey == "" {
		return nil, formErr(FormRealityKeyRequired, "Reality public key is required when Security is reality")
	}
	if p.Name == "" {
		p.Name = p.Server
		if p.Name == "" {
			p.Name = protocol
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	p.EnsureID()
	return p, nil
}
