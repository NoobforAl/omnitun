package importer

import (
	"fmt"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

var torTransports = []string{"obfs4", "webtunnel", "snowflake", "meek_lite"}

// torBuiltinKinds are the built-in bridge sets a tor:// link can name.
var torBuiltinKinds = []string{"obfs4", "snowflake"}

// bridgeLine returns line without a torrc "Bridge " prefix and reports whether
// it looks like a Tor bridge line: "<transport> <addr> ..." or a plain
// "IP:PORT FINGERPRINT".
func bridgeLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if rest, ok := strings.CutPrefix(line, "Bridge "); ok {
		line = strings.TrimSpace(rest)
	}
	f := strings.Fields(line)
	if len(f) < 2 {
		return "", false
	}
	for _, t := range torTransports {
		if f[0] == t {
			return line, true
		}
	}
	if strings.Contains(f[0], ":") && len(f[1]) == 40 && !strings.Contains(f[0], "/") {
		return line, true
	}
	return "", false
}

// parseBridgeLines turns pasted bridge lines (as from bridges.torproject.org
// or a torrc) into one Tor profile. It returns nil if text isn't all bridges.
func parseBridgeLines(text string) *profile.Profile {
	var bridges []string
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		b, ok := bridgeLine(line)
		if !ok {
			return nil
		}
		bridges = append(bridges, b)
	}
	if len(bridges) == 0 {
		return nil
	}
	p := &profile.Profile{
		Name:     "Tor (custom bridges)",
		Protocol: profile.Tor,
		Tor:      &profile.TorOpts{Bridges: bridges},
		Source:   "tor-bridges:" + strings.Join(bridges, "\n"),
	}
	p.EnsureID()
	return p
}

// parseTorLink parses tor:// (direct), tor://obfs4 and tor://snowflake
// (Tor Browser's built-in bridges).
func parseTorLink(link string) (*profile.Profile, error) {
	_, rest, _ := strings.Cut(link, "://")
	kind := strings.ToLower(strings.Trim(rest, "/ "))
	p := &profile.Profile{Name: "Tor", Protocol: profile.Tor, Tor: &profile.TorOpts{}}
	if kind != "" {
		ok := false
		for _, k := range torBuiltinKinds {
			ok = ok || k == kind
		}
		if !ok {
			return nil, fmt.Errorf("unknown Tor bridge kind %q (use obfs4 or snowflake)", kind)
		}
		p.Tor.Builtin = kind
		p.Name = "Tor (" + kind + ")"
	}
	return p, nil
}
