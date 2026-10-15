// Package serverconf turns the server state into a sing-box server config and
// into per-user share links that the Omnitun client (and other clients) import.
package serverconf

import (
	"encoding/base64"
	"fmt"
	"maps"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/NoobforAl/omnitun/server/store"
)

type obj = map[string]any

// Build returns the sing-box config for the active users. Paths are
// resolved against s.Dir. Protocols with no active users are left out
// (sing-box needs at least one user per inbound).
func Build(st *store.State, s *store.Store, now time.Time) obj {
	users := st.ActiveUsers(now)
	srv := st.Server
	var inbounds []any
	tls := func(extra obj) obj {
		t := obj{
			"enabled":          true,
			"certificate_path": s.Path(srv.CertFile),
			"key_path":         s.Path(srv.KeyFile),
		}
		maps.Copy(t, extra)
		return t
	}
	for _, name := range store.AllProtocols {
		p := srv.Protocols[name]
		if p == nil || !p.Enabled || len(users) == 0 {
			continue
		}
		in := obj{"tag": name, "listen": "::", "listen_port": p.Port}
		switch name {
		case store.VLESSReality:
			host, port := splitHostPort(srv.Reality.Handshake, 443)
			in["type"] = "vless"
			in["users"] = mapUsers(users, func(u *store.User) obj {
				return obj{"name": u.Name, "uuid": u.UUID, "flow": "xtls-rprx-vision"}
			})
			in["tls"] = obj{
				"enabled":     true,
				"server_name": host,
				"reality": obj{
					"enabled":     true,
					"handshake":   obj{"server": host, "server_port": port},
					"private_key": srv.Reality.PrivateKey,
					"short_id":    []string{srv.Reality.ShortID},
				},
			}
		case store.Shadowsocks:
			in["type"] = "shadowsocks"
			in["method"] = "2022-blake3-aes-128-gcm"
			in["password"] = srv.SSServerKey
			in["users"] = mapUsers(users, func(u *store.User) obj {
				return obj{"name": u.Name, "password": u.SSKey}
			})
		case store.Trojan:
			in["type"] = "trojan"
			in["users"] = mapUsers(users, func(u *store.User) obj {
				return obj{"name": u.Name, "password": u.Password}
			})
			in["tls"] = tls(nil)
		case store.Hysteria2:
			in["type"] = "hysteria2"
			in["users"] = mapUsers(users, func(u *store.User) obj {
				return obj{"name": u.Name, "password": u.Password}
			})
			in["tls"] = tls(obj{"alpn": []string{"h3"}})
		case store.TUIC:
			in["type"] = "tuic"
			in["congestion_control"] = "bbr"
			in["users"] = mapUsers(users, func(u *store.User) obj {
				return obj{"name": u.Name, "uuid": u.UUID, "password": u.Password}
			})
			in["tls"] = tls(obj{"alpn": []string{"h3"}})
		}
		inbounds = append(inbounds, in)
	}
	return obj{
		"log":       obj{"level": "warn", "timestamp": true},
		"inbounds":  inbounds,
		"outbounds": []any{obj{"type": "direct", "tag": "direct"}},
	}
}

func mapUsers(users []*store.User, f func(*store.User) obj) []any {
	out := make([]any, 0, len(users))
	for _, u := range users {
		out = append(out, f(u))
	}
	return out
}

func splitHostPort(hp string, defPort int) (string, int) {
	host, p, err := net.SplitHostPort(hp)
	if err != nil {
		return hp, defPort
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return host, defPort
	}
	return host, port
}

// Link is one share link for a user.
type Link struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// Links returns the user's share links for every enabled protocol, in
// failover-friendly order (Reality first, UDP protocols last).
func Links(st *store.State, u *store.User) []Link {
	srv := st.Server
	host := srv.PublicHost
	hostPort := func(port int) string { return net.JoinHostPort(host, strconv.Itoa(port)) }
	label := func(proto string) string { return url.PathEscape(srv.Name + " " + proto) }
	insecure := ""
	if srv.SelfSigned {
		insecure = "&insecure=1&allowInsecure=1&allow_insecure=1"
	}
	var out []Link
	for _, name := range store.AllProtocols {
		p := srv.Protocols[name]
		if p == nil || !p.Enabled {
			continue
		}
		var link string
		switch name {
		case store.VLESSReality:
			sni, _ := splitHostPort(srv.Reality.Handshake, 443)
			link = fmt.Sprintf("vless://%s@%s?type=tcp&security=reality&flow=xtls-rprx-vision&sni=%s&fp=chrome&pbk=%s&sid=%s#%s",
				u.UUID, hostPort(p.Port), sni, srv.Reality.PublicKey, srv.Reality.ShortID, label("VLESS-Reality"))
		case store.Shadowsocks:
			userinfo := base64.RawURLEncoding.EncodeToString([]byte("2022-blake3-aes-128-gcm:" + srv.SSServerKey + ":" + u.SSKey))
			link = fmt.Sprintf("ss://%s@%s#%s", userinfo, hostPort(p.Port), label("Shadowsocks"))
		case store.Trojan:
			link = fmt.Sprintf("trojan://%s@%s?security=tls&sni=%s%s#%s",
				u.Password, hostPort(p.Port), host, insecure, label("Trojan"))
		case store.Hysteria2:
			link = fmt.Sprintf("hysteria2://%s@%s?sni=%s%s#%s",
				u.Password, hostPort(p.Port), host, insecure, label("Hysteria2"))
		case store.TUIC:
			link = fmt.Sprintf("tuic://%s:%s@%s?congestion_control=bbr&alpn=h3&sni=%s%s#%s",
				u.UUID, u.Password, hostPort(p.Port), host, insecure, label("TUIC"))
		}
		out = append(out, Link{Protocol: name, URL: link})
	}
	return out
}

// SubscriptionURL is the user's subscription link, or "" if disabled.
func SubscriptionURL(st *store.State, u *store.User) string {
	sub := st.Server.Subscription
	if !sub.Enabled {
		return ""
	}
	return fmt.Sprintf("http://%s/%s/%s", net.JoinHostPort(st.Server.PublicHost, strconv.Itoa(sub.Port)), sub.Path, u.SubToken)
}
