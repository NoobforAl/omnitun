// Package importer turns share links, config files and subscription bodies
// into profiles.
package importer

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// ErrUnsupported is returned for input that no parser recognises.
var ErrUnsupported = errors.New("unsupported config format")

// ErrNoProfiles is returned when the text held links but none of them was valid.
var ErrNoProfiles = errors.New("no valid profiles")

// ErrXrayServer is returned for the config of an Xray server (not a client).
var ErrXrayServer = errors.New("this is an Xray server config; import it on the server (omnitun-server import xray), not as a client profile")

// UnsupportedError is a link of a scheme nothing here can read.
type UnsupportedError struct{ Scheme string }

func (e *UnsupportedError) Error() string { return ErrUnsupported.Error() + ": " + e.Scheme + "://" }

func (e *UnsupportedError) Unwrap() error { return ErrUnsupported }

// Result is the outcome of importing a blob of text.
type Result struct {
	Profiles []*profile.Profile `json:"profiles"`
	// Errors lists lines that looked like configs but failed to parse.
	Errors []string `json:"errors,omitempty"`
}

// Import detects the format of text and parses every profile it contains.
// Supported: share links (one per line), base64 subscription bodies and
// WireGuard/AmneziaWG .conf files.
func Import(text string) (*Result, error) {
	text = strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if text == "" {
		return nil, ErrUnsupported
	}
	if looksLikeJSON(text) {
		switch kindOfJSON(text) {
		case "sip008":
			return ImportSIP008(text)
		case "xray":
			return ImportXray(text)
		case "xray-server":
			return nil, ErrXrayServer
		}
		return ImportSingBoxJSON(text)
	}
	if looksLikeClash(text) {
		return ImportClash(text)
	}
	if looksLikeOVPN(text) {
		p, err := ParseOVPN(text)
		if err != nil {
			return nil, fmt.Errorf("openvpn: %w", err)
		}
		return &Result{Profiles: []*profile.Profile{p}}, nil
	}
	if looksLikeWireGuardConf(text) {
		p, err := ParseWireGuardConf(text)
		if err != nil {
			return nil, err
		}
		return &Result{Profiles: []*profile.Profile{p}}, nil
	}
	if p := parseBridgeLines(text); p != nil {
		return &Result{Profiles: []*profile.Profile{p}}, nil
	}
	if !strings.Contains(text, "://") {
		if decoded, ok := decodeBase64(text); ok && strings.Contains(decoded, "://") {
			text = decoded
		}
	}
	res := &Result{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		p, err := ParseLink(line)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", truncate(line, 48), err))
			continue
		}
		res.Profiles = append(res.Profiles, p)
	}
	if len(res.Profiles) == 0 {
		if len(res.Errors) > 0 {
			return res, fmt.Errorf("%w: %s", ErrNoProfiles, res.Errors[0])
		}
		return nil, ErrUnsupported
	}
	return res, nil
}

// ParseLink parses a single share link.
func ParseLink(link string) (*profile.Profile, error) {
	scheme, _, ok := strings.Cut(link, "://")
	if !ok {
		return nil, ErrUnsupported
	}
	var (
		p   *profile.Profile
		err error
	)
	switch strings.ToLower(scheme) {
	case "vless":
		p, err = parseVLESS(link)
	case "vmess":
		p, err = parseVMess(link)
	case "trojan":
		p, err = parseTrojan(link)
	case "ss":
		p, err = parseShadowsocks(link)
	case "hysteria2", "hy2":
		p, err = parseHysteria2(link)
	case "tuic":
		p, err = parseTUIC(link)
	case "wireguard", "wg", "awg":
		p, err = parseWireGuardLink(link)
	case "hysteria", "hy":
		p, err = parseHysteria(link)
	case "anytls":
		p, err = parseAnyTLS(link)
	case "ssh":
		p, err = parseSSH(link)
	case "naive+https", "naive+quic", "naive":
		p, err = parseNaive(link)
	case "ikev2":
		p, err = parseIKEv2(link)
	case "sstp":
		p, err = parseSSTP(link)
	case "tor":
		p, err = parseTorLink(link)
	default:
		return nil, &UnsupportedError{Scheme: scheme}
	}
	if err != nil {
		return nil, err
	}
	p.Source = link
	if err := p.Validate(); err != nil {
		return nil, err
	}
	p.EnsureID()
	return p, nil
}

// decodeBase64 accepts standard and URL alphabets, padded or not.
func decodeBase64(s string) (string, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), true
		}
	}
	return "", false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
