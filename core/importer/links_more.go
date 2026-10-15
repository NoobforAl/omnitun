package importer

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// hysteria://host:port?protocol=udp&auth=..&peer=sni&insecure=1&upmbps=10&downmbps=50&alpn=hysteria&obfsParam=..#name
func parseHysteria(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	p := &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.Hysteria,
		Server:   host,
		Port:     port,
		Password: firstNonEmpty(q.Get("auth"), q.Get("auth_str"), u.User.Username()),
		TLS: &profile.TLS{
			Enabled:    true,
			ServerName: firstNonEmpty(q.Get("peer"), q.Get("sni")),
			Insecure:   truthy(q.Get("insecure")) || truthy(q.Get("allowInsecure")),
			ALPN:       splitList(q.Get("alpn")),
		},
	}
	h := &profile.Hy2{Obfs: firstNonEmpty(q.Get("obfsParam"), q.Get("obfs-password"))}
	if q.Get("obfs") != "" && q.Get("obfs") != "xplus" && h.Obfs == "" {
		h.Obfs = q.Get("obfs")
	}
	h.UpMbps, _ = strconv.Atoi(firstNonEmpty(q.Get("upmbps"), q.Get("up")))
	h.DownMbps, _ = strconv.Atoi(firstNonEmpty(q.Get("downmbps"), q.Get("down")))
	p.Hysteria2 = h
	return p, nil
}

// anytls://password@host:port/?sni=..&insecure=1&fp=chrome#name
func parseAnyTLS(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	password, _ := url.PathUnescape(u.User.Username())
	return &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.AnyTLS,
		Server:   host,
		Port:     port,
		Password: password,
		TLS: &profile.TLS{
			Enabled:     true,
			ServerName:  firstNonEmpty(q.Get("sni"), q.Get("peer")),
			Insecure:    truthy(q.Get("insecure")) || truthy(q.Get("allowInsecure")),
			ALPN:        splitList(q.Get("alpn")),
			Fingerprint: q.Get("fp"),
		},
	}, nil
}

// ssh://user:password@host:port?pk=<PEM private key>&pkp=<passphrase>&hk=<host key>,<host key>#name
// (the format used by sing-box based clients such as Hiddify).
func parseSSH(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	password, _ := u.User.Password()
	o := &profile.SSHOpts{
		PrivateKey:           firstNonEmpty(q.Get("pk"), q.Get("private_key")),
		PrivateKeyPassphrase: firstNonEmpty(q.Get("pkp"), q.Get("private_key_passphrase")),
		HostKey:              splitList(firstNonEmpty(q.Get("hk"), q.Get("host_key"))),
		ClientVersion:        q.Get("client_version"),
	}
	p := &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.SSH,
		Server:   host,
		Port:     port,
		Username: u.User.Username(),
		Password: password,
	}
	if o.PrivateKey != "" || len(o.HostKey) > 0 || o.ClientVersion != "" {
		p.SSH = o
	}
	return p, nil
}

// naive+https://user:pass@host:port/?extra-headers=K%3AV%0D%0AK2%3AV2#name  (naive+quic:// for HTTP/3)
func parseNaive(link string) (*profile.Profile, error) {
	quic := strings.HasPrefix(strings.ToLower(link), "naive+quic://")
	_, rest, _ := strings.Cut(link, "://")
	u, host, port, err := parseURL("https://" + rest)
	if err != nil {
		if !strings.Contains(err.Error(), "invalid port") {
			return nil, err
		}
		// The port is optional (default 443).
		u, err = url.Parse("https://" + rest)
		if err != nil {
			return nil, err
		}
		host, port = u.Hostname(), 443
	}
	password, _ := u.User.Password()
	p := &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.Naive,
		Server:   host,
		Port:     port,
		Username: u.User.Username(),
		Password: password,
		TLS:      &profile.TLS{Enabled: true, ServerName: u.Query().Get("sni")},
	}
	o := &profile.NaiveOpts{QUIC: quic}
	if h := u.Query().Get("extra-headers"); h != "" {
		o.ExtraHeaders = map[string]string{}
		for line := range strings.SplitSeq(strings.ReplaceAll(h, "\r\n", "\n"), "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok {
				o.ExtraHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	if o.QUIC || len(o.ExtraHeaders) > 0 {
		p.Naive = o
	}
	return p, nil
}
