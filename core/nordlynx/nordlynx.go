// Package nordlynx turns a NordVPN access token into WireGuard profiles
// (NordLynx is WireGuard; the private key and servers come from NordVPN's API).
//
// Token: my.nordaccount.com → NordVPN → Access tokens.
package nordlynx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// BaseURL is NordVPN's API (overridden in tests).
var BaseURL = "https://api.nordvpn.com"

// Settings every NordLynx server shares.
const (
	address = "10.5.0.2/32"
	port    = 51820
)

var dns = []string{"103.86.96.100", "103.86.99.100"}

// ErrAuth means NordVPN rejected the access token.
var ErrAuth = errors.New("NordVPN rejected the access token")

type server struct {
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	Station   string `json:"station"`
	Locations []struct {
		Country struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"country"`
	} `json:"locations"`
	Technologies []struct {
		Identifier string `json:"identifier"`
		Metadata   []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"metadata"`
	} `json:"technologies"`
}

func (s server) publicKey() string {
	for _, t := range s.Technologies {
		if t.Identifier != "wireguard_udp" {
			continue
		}
		for _, m := range t.Metadata {
			if m.Name == "public_key" {
				return m.Value
			}
		}
	}
	return ""
}

func get(ctx context.Context, client *http.Client, path string, query url.Values, token string, out any) error {
	u := BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.SetBasicAuth("token", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("NordVPN API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrAuth
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NordVPN API %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

// countryID looks up NordVPN's numeric ID for a two-letter country code.
func countryID(ctx context.Context, client *http.Client, code string) (int, error) {
	var countries []struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	if err := get(ctx, client, "/v1/servers/countries", nil, "", &countries); err != nil {
		return 0, err
	}
	for _, c := range countries {
		if strings.EqualFold(c.Code, code) {
			return c.ID, nil
		}
	}
	return 0, fmt.Errorf("NordVPN has no servers in %q", code)
}

// Profiles fetches the user's NordLynx key and the recommended servers
// (optionally in one country, two-letter code) as WireGuard profiles.
func Profiles(ctx context.Context, client *http.Client, token, country string, limit int) ([]*profile.Profile, error) {
	if token == "" {
		return nil, errors.New("missing NordVPN access token")
	}
	if limit <= 0 {
		limit = 10
	}
	var creds struct {
		PrivateKey string `json:"nordlynx_private_key"`
	}
	if err := get(ctx, client, "/v1/users/services/credentials", nil, token, &creds); err != nil {
		return nil, err
	}
	if creds.PrivateKey == "" {
		return nil, errors.New("NordVPN returned no NordLynx key (is the subscription active?)")
	}
	q := url.Values{}
	q.Set("filters[servers_technologies][identifier]", "wireguard_udp")
	q.Set("limit", strconv.Itoa(limit))
	if country != "" {
		id, err := countryID(ctx, client, country)
		if err != nil {
			return nil, err
		}
		q.Set("filters[country_id]", strconv.Itoa(id))
	}
	var servers []server
	if err := get(ctx, client, "/v1/servers/recommendations", q, "", &servers); err != nil {
		return nil, err
	}
	var out []*profile.Profile
	for _, s := range servers {
		key := s.publicKey()
		if key == "" || s.Station == "" {
			continue
		}
		name := s.Name
		if name == "" {
			name = s.Hostname
		}
		p := &profile.Profile{
			Name: "NordLynx " + name, Protocol: profile.WireGuard, Server: s.Station, Port: port,
			WireGuard: &profile.WG{
				PrivateKey: creds.PrivateKey, PeerPublicKey: key,
				Address: []string{address}, DNS: dns, AllowedIPs: []string{"0.0.0.0/0", "::/0"},
			},
			Source: "nordlynx:" + s.Hostname,
		}
		p.EnsureID()
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("NordVPN returned no WireGuard servers")
	}
	return out, nil
}
