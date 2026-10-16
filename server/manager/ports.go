package manager

import (
	"fmt"
	"net"
	"sort"

	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/store"
)

// PortRule is a port the firewall must let through for the server to work.
type PortRule struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto" enum:"tcp,udp"`
	Purpose string `json:"purpose" doc:"What needs it, e.g. the protocol key"`
}

// FirewallPorts lists every port the current configuration needs open:
// each enabled protocol on its transports, the subscription endpoint, the
// ACME challenge port when a real certificate is requested without DNS-01,
// and the web API when it listens beyond the loopback interface.
func (m *Manager) FirewallPorts() ([]PortRule, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	var rules []PortRule
	add := func(port int, proto, purpose string) {
		rules = append(rules, PortRule{Port: port, Proto: proto, Purpose: purpose})
	}
	for _, key := range store.AllProtocols {
		p := st.Server.Protocols[key]
		proto, ok := protocols.Get(key)
		if p == nil || !p.Enabled || !ok {
			continue
		}
		switch proto.Spec().Transport {
		case "tcp":
			add(p.Port, "tcp", key)
		case "udp":
			add(p.Port, "udp", key)
		default:
			add(p.Port, "tcp", key)
			add(p.Port, "udp", key)
		}
	}
	if sub := st.Server.Subscription; sub.Enabled {
		add(sub.Port, "tcp", "subscriptions")
	}
	if a := st.Server.ACME; st.Server.ACMEActive() && a.DNSProvider == "" {
		port := 80
		if a.HTTPPort > 0 {
			port = a.HTTPPort
		}
		add(port, "tcp", "acme-http-challenge")
	}
	if a := st.Server.API; a.Enabled && !isLoopback(a.Listen) {
		add(a.Port, "tcp", "web-api")
	}
	return dedupe(rules), nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// dedupe merges rules for the same port and protocol, joining their purposes.
func dedupe(rules []PortRule) []PortRule {
	type key struct {
		port  int
		proto string
	}
	seen := map[key]int{}
	var out []PortRule
	for _, r := range rules {
		k := key{r.Port, r.Proto}
		if i, ok := seen[k]; ok {
			out[i].Purpose = fmt.Sprintf("%s, %s", out[i].Purpose, r.Purpose)
			continue
		}
		seen[k] = len(out)
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Proto != out[j].Proto {
			return out[i].Proto < out[j].Proto
		}
		return out[i].Port < out[j].Port
	})
	return out
}
