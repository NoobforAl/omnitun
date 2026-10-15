package importer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// looksLikeOVPN reports whether text is an OpenVPN client config.
func looksLikeOVPN(text string) bool {
	hasRemote, hasClientish := false, false
	for line := range strings.SplitSeq(text, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch strings.ToLower(f[0]) {
		case "remote":
			hasRemote = true
		case "client", "dev", "<ca>", "proto", "tls-client", "pull":
			hasClientish = true
		}
	}
	return hasRemote && hasClientish
}

// ParseOVPN parses a single-file OpenVPN client config (certificates inline).
// Client-side tuning directives that don't change the connection are ignored.
func ParseOVPN(text string) (*profile.Profile, error) {
	o := &profile.OVPN{}
	p := &profile.Profile{Protocol: profile.OpenVPN, OpenVPN: o, Source: text}
	defaultPort, defaultProto := 1194, "udp"
	type remote struct {
		host  string
		port  int
		proto string
	}
	var remotes []remote
	var unsupported []string

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		// Inline blocks: <ca> ... </ca>
		if strings.HasPrefix(line, "<") && strings.HasSuffix(line, ">") && !strings.HasPrefix(line, "</") {
			tag := strings.Trim(line, "<>")
			var body []string
			for i++; i < len(lines) && strings.TrimSpace(lines[i]) != "</"+tag+">"; i++ {
				body = append(body, lines[i])
			}
			if i >= len(lines) {
				return nil, fmt.Errorf("unterminated <%s> block", tag)
			}
			content := strings.TrimSpace(strings.Join(body, "\n"))
			switch tag {
			case "ca":
				o.CA = content
			case "cert":
				o.Cert = content
			case "key":
				o.Key = content
			case "tls-auth", "tls-crypt", "tls-crypt-v2":
				o.ControlWrap, o.ControlWrapKey = tag, content
			case "auth-user-pass":
				creds := strings.SplitN(content, "\n", 2)
				if len(creds) == 2 {
					p.Username, p.Password = strings.TrimSpace(creds[0]), strings.TrimSpace(creds[1])
				}
			case "pkcs12", "secret", "http-proxy-user-pass":
				unsupported = append(unsupported, "<"+tag+">")
			}
			continue
		}

		f := strings.Fields(line)
		arg := func(n int) string {
			if n < len(f) {
				return strings.Trim(f[n], `"`)
			}
			return ""
		}
		switch dir := strings.ToLower(f[0]); dir {
		case "remote":
			r := remote{host: arg(1), port: -1, proto: arg(3)}
			if v := arg(2); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil || n <= 0 || n > 65535 {
					return nil, fmt.Errorf("invalid remote port %q", v)
				}
				r.port = n
			}
			remotes = append(remotes, r)
		case "port", "rport":
			if n, err := strconv.Atoi(arg(1)); err == nil {
				defaultPort = n
			}
		case "proto":
			defaultProto = arg(1)
		case "dev":
			if strings.HasPrefix(arg(1), "tap") {
				return nil, fmt.Errorf("TAP (layer 2) OpenVPN configs are not supported, only TUN")
			}
		case "remote-random":
			o.RemoteRandom = true
		case "cipher":
			o.Cipher = arg(1)
		case "data-ciphers", "ncp-ciphers":
			o.DataCiphers = strings.Split(arg(1), ":")
		case "data-ciphers-fallback":
			o.DataCiphersFallback = arg(1)
		case "auth":
			o.Auth = arg(1)
		case "comp-lzo":
			o.CompressionLZO = orDefault(arg(1), "adaptive")
		case "compress":
			o.Compression = orDefault(arg(1), "stub")
		case "remote-cert-tls":
			o.RemoteCertTLS = arg(1)
		case "verify-x509-name":
			o.VerifyX509Name, o.VerifyX509Type = arg(1), arg(2)
		case "tls-version-min":
			o.TLSVersionMin = arg(1)
		case "tls-cipher":
			o.TLSCipher = arg(1)
		case "key-direction":
			o.KeyDirection = arg(1)
		case "tun-mtu":
			o.MTU, _ = strconv.Atoi(arg(1))
		case "mssfix":
			o.MSSFix, _ = strconv.Atoi(arg(1))
		case "auth-user-pass":
			if arg(1) != "" {
				return nil, fmt.Errorf("auth-user-pass refers to the file %q; enter the username and password in the app instead", arg(1))
			}
			o.NeedsAuth = true
		case "tls-auth", "tls-crypt", "tls-crypt-v2":
			if arg(1) != "" && arg(1) != "[inline]" {
				return nil, fmt.Errorf("%s refers to the file %q; use a single-file .ovpn with the key inline", dir, arg(1))
			}
			if dir == "tls-auth" && arg(2) != "" {
				o.KeyDirection = arg(2)
			}
		case "ca", "cert", "key", "pkcs12":
			if arg(1) != "[inline]" {
				return nil, fmt.Errorf("%s refers to the file %q; use a single-file .ovpn with certificates inline", dir, arg(1))
			}
		case "secret":
			return nil, fmt.Errorf("static-key OpenVPN configs are not supported")
		case "http-proxy", "socks-proxy":
			unsupported = append(unsupported, dir)
		}
	}
	if len(unsupported) > 0 {
		return nil, fmt.Errorf("unsupported OpenVPN options: %s", strings.Join(unsupported, ", "))
	}
	if len(remotes) == 0 {
		return nil, fmt.Errorf("no remote server")
	}
	if o.CA == "" {
		return nil, fmt.Errorf("no <ca> certificate")
	}
	if (o.Cert == "") != (o.Key == "") {
		return nil, fmt.Errorf("client certificate and key must both be present")
	}
	if o.Cert == "" && !o.NeedsAuth && p.Username == "" {
		return nil, fmt.Errorf("no client certificate and no auth-user-pass")
	}
	if p.Username != "" {
		o.NeedsAuth = true // inline credentials were given
	}
	for i, r := range remotes {
		if r.port < 0 {
			r.port = defaultPort
		}
		proto, err := ovpnNetwork(orDefault(r.proto, defaultProto))
		if err != nil {
			return nil, err
		}
		if i == 0 {
			p.Server, p.Port, o.Network = r.host, r.port, proto
			continue
		}
		o.Remotes = append(o.Remotes, profile.OVPNRemote{Server: r.host, Port: r.port, Network: proto})
	}
	p.Name = "OpenVPN " + p.Server
	if err := p.Validate(); err != nil {
		return nil, err
	}
	p.EnsureID()
	return p, nil
}

func ovpnNetwork(proto string) (string, error) {
	switch strings.ToLower(proto) {
	case "", "udp", "udp4", "udp6":
		return "udp", nil
	case "tcp", "tcp-client", "tcp4", "tcp6", "tcp4-client", "tcp6-client":
		return "tcp", nil
	}
	return "", fmt.Errorf("unsupported proto %q", proto)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
