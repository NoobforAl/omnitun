// Package subscription downloads and parses provider subscription links:
// a URL (optionally authenticated) that returns a list of share links,
// usually base64-encoded, plus usage/expiry headers.
package subscription

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/nordlynx"
)

// DefaultUserAgent asks panels for the plain share-link list most of them
// serve to v2rayN-compatible clients.
const DefaultUserAgent = "Omnitun/0.1 v2rayN/7.0"

const maxBody = 10 << 20

// Auth types.
const (
	AuthNone   = "none"
	AuthBasic  = "basic"
	AuthBearer = "bearer"
	AuthHeader = "header" // custom header name + value
	AuthQuery  = "query"  // token added as a URL query parameter
)

// Auth describes how to authenticate the request.
type Auth struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	Header   string `json:"header,omitempty"` // header name for AuthHeader
	QueryKey string `json:"query_key,omitempty"`
}

// Request is one subscription to fetch.
type Request struct {
	URL       string `json:"url"`
	Auth      Auth   `json:"auth"`
	UserAgent string `json:"user_agent,omitempty"`
}

// Info is metadata from the provider's response headers.
type Info struct {
	Upload   int64 `json:"upload,omitempty"`   // bytes
	Download int64 `json:"download,omitempty"` // bytes
	Total    int64 `json:"total,omitempty"`    // bytes, 0 = unlimited/unknown
	// Expire is a Unix timestamp, 0 = never/unknown.
	Expire int64 `json:"expire,omitempty"`
	// UpdateIntervalHours is the provider's suggested refresh interval.
	UpdateIntervalHours int    `json:"update_interval_hours,omitempty"`
	Title               string `json:"title,omitempty"`
}

// Result of a fetch.
type Result struct {
	importer.Result
	Info      Info      `json:"info"`
	FetchedAt time.Time `json:"fetched_at"`
}

// ErrAuth is returned for 401/403 responses.
var ErrAuth = errors.New("subscription authentication failed")

// Fetch downloads and parses a subscription. rootCAs may be nil to use the
// system pool (Android needs an explicit pool, see engine.Config.CertificateStore).
func Fetch(ctx context.Context, req Request, rootCAs *x509.CertPool) (*Result, error) {
	u, err := url.Parse(strings.TrimSpace(req.URL))
	if err == nil && u.Scheme == "nordvpn" {
		return fetchNordVPN(ctx, u, req, rootCAs)
	}
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid subscription URL %q", req.URL)
	}
	if req.Auth.Type == AuthQuery {
		key := req.Auth.QueryKey
		if key == "" {
			key = "token"
		}
		q := u.Query()
		q.Set(key, req.Auth.Token)
		u.RawQuery = q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	ua := req.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	httpReq.Header.Set("User-Agent", ua)
	switch req.Auth.Type {
	case "", AuthNone, AuthQuery:
	case AuthBasic:
		httpReq.SetBasicAuth(req.Auth.Username, req.Auth.Password)
	case AuthBearer:
		httpReq.Header.Set("Authorization", "Bearer "+req.Auth.Token)
	case AuthHeader:
		if req.Auth.Header == "" {
			return nil, errors.New("custom header auth needs a header name")
		}
		httpReq.Header.Set(req.Auth.Header, req.Auth.Token)
	default:
		return nil, fmt.Errorf("unknown auth type %q", req.Auth.Type)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if rootCAs != nil {
		transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("download subscription: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w (HTTP %d)", ErrAuth, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("subscription server returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("read subscription: %w", err)
	}
	if len(body) > maxBody {
		return nil, errors.New("subscription is larger than 10 MB")
	}

	parsed, err := importer.Import(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse subscription: %w", err)
	}
	return &Result{Result: *parsed, Info: parseInfo(resp.Header), FetchedAt: time.Now()}, nil
}

// parseInfo reads the de-facto standard headers:
// subscription-userinfo: upload=1; download=2; total=3; expire=4
// profile-update-interval: 24
// profile-title: name or base64:bmFtZQ==
func parseInfo(h http.Header) Info {
	var info Info
	for part := range strings.SplitSeq(h.Get("Subscription-Userinfo"), ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(k) {
		case "upload":
			info.Upload = n
		case "download":
			info.Download = n
		case "total":
			info.Total = n
		case "expire":
			info.Expire = n
		}
	}
	if n, err := strconv.Atoi(strings.TrimSpace(h.Get("Profile-Update-Interval"))); err == nil && n > 0 {
		info.UpdateIntervalHours = n
	}
	title := strings.TrimSpace(h.Get("Profile-Title"))
	if b64, ok := strings.CutPrefix(title, "base64:"); ok {
		if dec, err := base64.StdEncoding.DecodeString(b64); err == nil {
			title = string(dec)
		}
	}
	info.Title = title
	return info
}

// fetchNordVPN handles nordvpn://<access token>?country=DE&limit=10: NordLynx
// (WireGuard) profiles from NordVPN's API. The token may also be given as
// bearer/basic auth instead of in the URL.
func fetchNordVPN(ctx context.Context, u *url.URL, req Request, rootCAs *x509.CertPool) (*Result, error) {
	token := firstNonEmptyStr(u.Host, u.User.Username(), req.Auth.Token, req.Auth.Password)
	limit, _ := strconv.Atoi(u.Query().Get("limit"))
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if rootCAs != nil {
		transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()
	profiles, err := nordlynx.Profiles(ctx, client, token, u.Query().Get("country"), limit)
	if errors.Is(err, nordlynx.ErrAuth) {
		return nil, fmt.Errorf("%w: %v", ErrAuth, err)
	}
	if err != nil {
		return nil, err
	}
	return &Result{
		Profiles:  profiles,
		Info:      Info{Title: "NordVPN", UpdateIntervalHours: 24},
		FetchedAt: time.Now(),
	}, nil
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
