package importer

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// kindOfJSON tells Xray/V2Ray configs, SIP008 server lists and sing-box
// configs apart by looking at their top-level shape.
func kindOfJSON(text string) string {
	var doc struct {
		Servers   []map[string]json.RawMessage `json:"servers"`
		Version   json.RawMessage              `json:"version"`
		Outbounds []map[string]json.RawMessage `json:"outbounds"`
		Inbounds  []map[string]json.RawMessage `json:"inbounds"`
	}
	if json.Unmarshal([]byte(text), &doc) != nil {
		return "sing-box"
	}
	if len(doc.Servers) > 0 && doc.Servers[0]["server"] != nil {
		return "sip008"
	}
	proxyProtocols := map[string]bool{"vless": true, "vmess": true, "trojan": true, "shadowsocks": true, "wireguard": true, "hysteria": true}
	protocolOf := func(m map[string]json.RawMessage) string {
		var v string
		json.Unmarshal(m["protocol"], &v)
		return strings.ToLower(v)
	}
	proxyOutbound := false
	for _, o := range doc.Outbounds {
		if o["type"] == nil && proxyProtocols[protocolOf(o)] {
			proxyOutbound = true
		}
	}
	proxyInbound := false
	for _, in := range doc.Inbounds {
		if proxyProtocols[protocolOf(in)] {
			proxyInbound = true
		}
	}
	switch {
	case proxyOutbound:
		return "xray"
	case proxyInbound:
		return "xray-server"
	}
	return "sing-box"
}

// ImportSIP008 reads a SIP008 Shadowsocks server list
// ({"version":1,"servers":[{"server","server_port","password","method","remarks",...}]}).
func ImportSIP008(text string) (*Result, error) {
	var doc struct {
		Servers []struct {
			ID         string `json:"id"`
			Remarks    string `json:"remarks"`
			Server     string `json:"server"`
			ServerPort int    `json:"server_port"`
			Password   string `json:"password"`
			Method     string `json:"method"`
			Plugin     string `json:"plugin"`
			PluginOpts string `json:"plugin_opts"`
		} `json:"servers"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("SIP008: %w", err)
	}
	res := &Result{}
	for _, s := range doc.Servers {
		p := &profile.Profile{
			Name: s.Remarks, Protocol: profile.Shadowsocks, Server: s.Server, Port: s.ServerPort,
			Method: s.Method, Password: s.Password, Plugin: s.Plugin, PluginOpts: s.PluginOpts,
			Source: "sip008:" + s.ID + ":" + s.Server + ":" + strconv.Itoa(s.ServerPort),
		}
		if err := p.Validate(); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		p.EnsureID()
		res.Profiles = append(res.Profiles, p)
	}
	if len(res.Profiles) == 0 {
		return res, fmt.Errorf("no valid servers in the SIP008 list")
	}
	return res, nil
}

type xrayStream struct {
	Network  string `json:"network"`
	Security string `json:"security"`
	TLS      struct {
		ServerName    string   `json:"serverName"`
		AllowInsecure bool     `json:"allowInsecure"`
		ALPN          []string `json:"alpn"`
		Fingerprint   string   `json:"fingerprint"`
	} `json:"tlsSettings"`
	Reality struct {
		ServerName  string `json:"serverName"`
		Fingerprint string `json:"fingerprint"`
		PublicKey   string `json:"publicKey"`
		ShortID     string `json:"shortId"`
	} `json:"realitySettings"`
	WS struct {
		Path    string            `json:"path"`
		Host    string            `json:"host"`
		Headers map[string]string `json:"headers"`
	} `json:"wsSettings"`
	HTTPUpgrade struct {
		Path string `json:"path"`
		Host string `json:"host"`
	} `json:"httpupgradeSettings"`
	GRPC struct {
		ServiceName string `json:"serviceName"`
	} `json:"grpcSettings"`
	HTTP struct {
		Path string   `json:"path"`
		Host []string `json:"host"`
	} `json:"httpSettings"`
}

// xrayStreamTo applies streamSettings to p.
func xrayStreamTo(p *profile.Profile, st *xrayStream) error {
	if st == nil {
		return nil
	}
	switch st.Security {
	case "tls":
		p.TLS = &profile.TLS{
			Enabled: true, ServerName: st.TLS.ServerName, Insecure: st.TLS.AllowInsecure,
			ALPN: st.TLS.ALPN, Fingerprint: st.TLS.Fingerprint,
		}
	case "reality":
		p.TLS = &profile.TLS{
			Enabled: true, ServerName: st.Reality.ServerName, Fingerprint: firstNonEmpty(st.Reality.Fingerprint, "chrome"),
			Reality: &profile.Reality{PublicKey: st.Reality.PublicKey, ShortID: st.Reality.ShortID},
		}
	}
	switch st.Network {
	case "", "tcp", "raw":
	case "ws":
		host := firstNonEmpty(st.WS.Host, st.WS.Headers["Host"], st.WS.Headers["host"])
		p.Transport = &profile.Transport{Type: "ws", Path: st.WS.Path, Host: host}
	case "httpupgrade":
		p.Transport = &profile.Transport{Type: "httpupgrade", Path: st.HTTPUpgrade.Path, Host: st.HTTPUpgrade.Host}
	case "grpc":
		p.Transport = &profile.Transport{Type: "grpc", ServiceName: st.GRPC.ServiceName}
	case "h2", "http":
		host := ""
		if len(st.HTTP.Host) > 0 {
			host = st.HTTP.Host[0]
		}
		p.Transport = &profile.Transport{Type: "http", Path: st.HTTP.Path, Host: host}
	default:
		return fmt.Errorf("transport %q is not supported", st.Network)
	}
	return nil
}

type xrayOutbound struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Settings struct {
		// Classic layout.
		VNext []struct {
			Address string `json:"address"`
			Port    int    `json:"port"`
			Users   []struct {
				ID       string `json:"id"`
				Flow     string `json:"flow"`
				AlterID  int    `json:"alterId"`
				Security string `json:"security"`
			} `json:"users"`
		} `json:"vnext"`
		Servers []struct {
			Address  string `json:"address"`
			Port     int    `json:"port"`
			Password string `json:"password"`
			Method   string `json:"method"`
		} `json:"servers"`
		// "address" is a host in the flat layout used by newer Xray
		// versions and a list of tunnel addresses for WireGuard.
		Address json.RawMessage `json:"address"`
		Port    int             `json:"port"`
		ID      string          `json:"id"`
		Flow    string          `json:"flow"`
		// WireGuard.
		SecretKey string `json:"secretKey"`
		MTU       int    `json:"mtu"`
		Peers     []struct {
			PublicKey    string `json:"publicKey"`
			PreSharedKey string `json:"preSharedKey"`
			Endpoint     string `json:"endpoint"`
		} `json:"peers"`
	} `json:"settings"`
	Stream *xrayStream `json:"streamSettings"`
}

// ImportXray reads the outbounds of an Xray/V2Ray client config (vless,
// vmess, trojan, shadowsocks, wireguard). Other outbounds (freedom,
// blackhole, dns, socks, http) are routing helpers and skipped.
func ImportXray(text string) (*Result, error) {
	var doc struct {
		Remarks   string            `json:"remarks"`
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("Xray JSON: %w", err)
	}
	res := &Result{}
	add := func(p *profile.Profile, ob xrayOutbound) {
		if p.Name == "" {
			p.Name = firstNonEmpty(ob.Tag, doc.Remarks, net.JoinHostPort(p.Server, strconv.Itoa(p.Port)))
		}
		if err := xrayStreamTo(p, ob.Stream); err != nil {
			res.Errors = append(res.Errors, p.Name+": "+err.Error())
			return
		}
		p.Source = "xray:" + p.Protocol + ":" + p.Server + ":" + strconv.Itoa(p.Port) + ":" + p.UUID + p.Password + ob.Tag
		if err := p.Validate(); err != nil {
			res.Errors = append(res.Errors, err.Error())
			return
		}
		p.EnsureID()
		res.Profiles = append(res.Profiles, p)
	}
	for _, raw := range doc.Outbounds {
		var ob xrayOutbound
		if err := json.Unmarshal(raw, &ob); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		s := ob.Settings
		switch strings.ToLower(ob.Protocol) {
		case "vless":
			if host := jsonString(s.Address); len(s.VNext) == 0 && host != "" {
				add(&profile.Profile{Protocol: profile.VLESS, Server: host, Port: s.Port, UUID: s.ID, Flow: s.Flow}, ob)
			}
			for _, v := range s.VNext {
				for _, u := range v.Users {
					add(&profile.Profile{Protocol: profile.VLESS, Server: v.Address, Port: v.Port, UUID: u.ID, Flow: u.Flow}, ob)
				}
			}
		case "vmess":
			for _, v := range s.VNext {
				for _, u := range v.Users {
					add(&profile.Profile{Protocol: profile.VMess, Server: v.Address, Port: v.Port, UUID: u.ID,
						AlterID: u.AlterID, Method: firstNonEmpty(u.Security, "auto")}, ob)
				}
			}
		case "trojan":
			for _, sv := range s.Servers {
				add(&profile.Profile{Protocol: profile.Trojan, Server: sv.Address, Port: sv.Port, Password: sv.Password}, ob)
			}
		case "shadowsocks":
			for _, sv := range s.Servers {
				add(&profile.Profile{Protocol: profile.Shadowsocks, Server: sv.Address, Port: sv.Port,
					Method: sv.Method, Password: sv.Password}, ob)
			}
		case "wireguard":
			for _, peer := range s.Peers {
				host, port := splitEndpoint(peer.Endpoint)
				add(&profile.Profile{Protocol: profile.WireGuard, Server: host, Port: port, WireGuard: &profile.WG{
					PrivateKey: s.SecretKey, PeerPublicKey: peer.PublicKey, PreSharedKey: peer.PreSharedKey,
					Address: jsonStrings(s.Address), MTU: s.MTU,
				}}, ob)
			}
		}
	}
	if len(res.Profiles) == 0 {
		if len(res.Errors) > 0 {
			return res, fmt.Errorf("no valid outbounds: %s", res.Errors[0])
		}
		return res, fmt.Errorf("no usable outbounds in the Xray config")
	}
	return res, nil
}

func splitEndpoint(ep string) (string, int) {
	host, port, err := net.SplitHostPort(ep)
	if err != nil {
		return ep, 0
	}
	n, _ := strconv.Atoi(port)
	return host, n
}

func jsonString(raw json.RawMessage) string {
	var v string
	json.Unmarshal(raw, &v)
	return v
}

func jsonStrings(raw json.RawMessage) []string {
	var v []string
	if json.Unmarshal(raw, &v) != nil {
		if one := jsonString(raw); one != "" {
			return []string{one}
		}
	}
	return v
}
