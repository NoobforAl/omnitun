package control

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/core/subscription"
)

var countryCode = regexp.MustCompile(`^[A-Z]{2}$`)

// DefaultExitInfoURL is asked through the VPN for the address the world sees
// and its country. It answers {"ip":"203.0.113.5","country":"DE"}.
const DefaultExitInfoURL = "https://api.country.is/"

// ExitInfoParams asks where the running tunnel exits.
type ExitInfoParams struct {
	// URL overrides DefaultExitInfoURL (any service that answers JSON with an
	// address and a country code).
	URL string `json:"url,omitempty"`
}

// ExitInfo is what the service saw: the tunnel's exit address and country.
type ExitInfo struct {
	IP string `json:"ip"`
	// Country is an ISO 3166-1 alpha-2 code, empty when the service did not say.
	Country string `json:"country,omitempty"`
}

// exitInfo asks the service through the running tunnel. It fails when no
// tunnel that can carry the request is running, so the answer is always about
// the tunnel's exit and never about this device's own address.
func (s *Server) exitInfo(ctx context.Context, p ExitInfoParams) (*ExitInfo, error) {
	dial, err := s.engine.TunnelDialer()
	if err != nil {
		return nil, err
	}
	target := strings.TrimSpace(p.URL)
	if target == "" {
		target = DefaultExitInfoURL
	}
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid address %q", target)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // Tor builds circuits slowly
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", subscription.DefaultUserAgent)
	client := &http.Client{Transport: s.transportVia(dial)}
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the service answered HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	return parseExitInfo(body)
}

// transportVia is an HTTP transport that opens its connections with dial.
func (s *Server) transportVia(dial subscription.Dialer) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = dial
	if pool := s.rootPool(); pool != nil {
		t.TLSClientConfig = &tls.Config{RootCAs: pool}
	}
	return t
}

// parseExitInfo reads the answers of the common "what is my IP" services:
// {"ip","country"} (country.is), {"ip","country_code"} (ipwho.is, ipapi),
// {"IP","IsTor"} (check.torproject.org, no country).
func parseExitInfo(body []byte) (*ExitInfo, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.New("the service did not answer with JSON")
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	info := &ExitInfo{IP: str("ip", "IP", "query")}
	if info.IP == "" {
		return nil, errors.New("the service did not name an address")
	}
	// A code is two letters; services also answer "T1" for Tor exits or "XX" for unknown.
	if c := strings.ToUpper(str("country", "country_code", "countryCode", "Country")); countryCode.MatchString(c) && c != "XX" {
		info.Country = c
	}
	return info, nil
}
