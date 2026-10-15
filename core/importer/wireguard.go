package importer

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// amneziaKeys are the AmneziaWG obfuscation fields in the [Interface] section.
var amneziaKeys = map[string]bool{
	"jc": true, "jmin": true, "jmax": true, "s1": true, "s2": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "itime": true,
	// AmneziaWG 2.0 (captured so they can be reported if unsupported)
	"s3": true, "s4": true, "i1": true, "i2": true, "i3": true, "i4": true, "i5": true,
	"j1": true, "j2": true, "j3": true,
}

func looksLikeWireGuardConf(text string) bool {
	return strings.Contains(text, "[Interface]") && strings.Contains(text, "[Peer]")
}

// ParseWireGuardConf parses a wg-quick style .conf file. AmneziaWG files
// (Jc/Jmin/Jmax/S1/S2/H1-H4 in [Interface]) produce an amneziawg profile.
// Only the first [Peer] is used.
func ParseWireGuardConf(text string) (*profile.Profile, error) {
	wg := &profile.WG{}
	p := &profile.Profile{Protocol: profile.WireGuard, WireGuard: wg}
	section := ""
	peers := 0
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.IndexAny(line, "#;"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			if section == "peer" {
				peers++
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("wireguard: malformed line %q", line)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch {
		case section == "interface":
			switch {
			case key == "privatekey":
				wg.PrivateKey = value
			case key == "address":
				wg.Address = append(wg.Address, splitList(value)...)
			case key == "dns":
				wg.DNS = append(wg.DNS, splitList(value)...)
			case key == "mtu":
				wg.MTU, _ = strconv.Atoi(value)
			case amneziaKeys[key]:
				if wg.Amnezia == nil {
					wg.Amnezia = map[string]string{}
				}
				wg.Amnezia[key] = value
			}
		case section == "peer" && peers == 1:
			switch key {
			case "publickey":
				wg.PeerPublicKey = value
			case "presharedkey":
				wg.PreSharedKey = value
			case "allowedips":
				wg.AllowedIPs = append(wg.AllowedIPs, splitList(value)...)
			case "persistentkeepalive":
				wg.Keepalive, _ = strconv.Atoi(value)
			case "endpoint":
				host, port, err := net.SplitHostPort(value)
				if err != nil {
					return nil, fmt.Errorf("wireguard: invalid endpoint %q", value)
				}
				p.Server = host
				p.Port, err = strconv.Atoi(port)
				if err != nil {
					return nil, fmt.Errorf("wireguard: invalid endpoint port %q", port)
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(wg.Amnezia) > 0 {
		p.Protocol = profile.AmneziaWG
	}
	p.Source = text
	if err := p.Validate(); err != nil {
		return nil, err
	}
	p.EnsureID()
	return p, nil
}
