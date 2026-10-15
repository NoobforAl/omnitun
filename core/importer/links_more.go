package importer

import (
	"net/url"
	"strconv"

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
