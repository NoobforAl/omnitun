package tor

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
)

// builtinJSON is Tor Browser's built-in bridge list (tor-browser-build
// projects/tor-expert-bundle/pt_config.json).
//
//go:embed builtin_bridges.json
var builtinJSON []byte

// Builtin returns Tor Browser's built-in bridges of a kind ("obfs4",
// "snowflake", "meek"); nil for an unknown kind.
func Builtin(kind string) []string {
	var cfg struct {
		Bridges map[string][]string `json:"bridges"`
	}
	if json.Unmarshal(builtinJSON, &cfg) != nil {
		return nil
	}
	return cfg.Bridges[kind]
}

// BuiltinKinds lists the built-in bridge kinds.
func BuiltinKinds() []string { return []string{"obfs4", "snowflake", "meek"} }

// NormalizeBridge trims a bridge line and drops a torrc "Bridge " prefix.
func NormalizeBridge(line string) string {
	line = strings.TrimSpace(line)
	if rest, ok := strings.CutPrefix(line, "Bridge "); ok {
		line = strings.TrimSpace(rest)
	}
	return line
}

// Transport returns the pluggable transport a bridge line uses, or "" for a
// plain (vanilla) bridge "IP:PORT FINGERPRINT".
func Transport(line string) string {
	first, _, _ := strings.Cut(NormalizeBridge(line), " ")
	if first == "" || strings.ContainsAny(first, ":.[") {
		return ""
	}
	return first
}

// IsBridgeLine reports whether line looks like a Tor bridge line.
func IsBridgeLine(line string) bool {
	f := strings.Fields(NormalizeBridge(line))
	if len(f) < 2 {
		return false
	}
	if t := Transport(line); t != "" {
		return slices.Contains(Supported, t) && strings.Contains(f[1], ":")
	}
	// Vanilla bridge: "IP:PORT FINGERPRINT".
	return strings.Contains(f[0], ":") && len(f[1]) == 40
}

// TransportsFor lists the distinct transports needed by bridges.
func TransportsFor(bridges []string) []string {
	var out []string
	for _, b := range bridges {
		if t := Transport(b); t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}
