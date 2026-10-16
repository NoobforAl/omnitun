// Package api is the server's management web API: JSON over HTTP with an
// OpenAPI 3 description (/openapi.json) and interactive docs (/docs). Every
// endpoint is a thin wrapper over server/manager and needs the bearer token.
package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/NoobforAl/omnitun/server/manager"
)

// Options configures the API.
type Options struct {
	Manager *manager.Manager
	// Token returns the bearer token every /api/ request must carry. It is
	// called per request, so a rotated token revokes the old one at once;
	// returning "" rejects every request (API disabled).
	Token func() string
	// Reload restarts the VPN core so changes apply at once; the running
	// server also reloads by itself within seconds. Nil when no server runs
	// in this process.
	Reload func() error
	// Version is shown in the OpenAPI document.
	Version string
}

// Health is the liveness response.
type Health struct {
	OK bool `json:"ok"`
}

type secured = []map[string][]string

var bearer = secured{{"bearer": {}}}

// toHTTP maps manager errors to HTTP errors.
func toHTTP(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, manager.ErrNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, manager.ErrInvalid):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, manager.ErrConflict):
		return huma.Error409Conflict(err.Error())
	}
	return huma.Error500InternalServerError(err.Error())
}

// New returns the API handler.
func New(o Options) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	version := o.Version
	if version == "" {
		version = "1.0.0"
	}
	cfg := huma.DefaultConfig("Omnitun Server API", version)
	cfg.Info.Description = "Manage an Omnitun server: protocols, users and share links. " +
		"Send the token printed by `omnitun-server api token` as `Authorization: Bearer <token>`."
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}
	a := humachi.New(router, cfg)
	a.UseMiddleware(authenticate(a, o.Token))

	m := o.Manager
	type body[T any] struct{ Body T }

	huma.Register(a, huma.Operation{
		OperationID: "health", Method: http.MethodGet, Path: "/health", Summary: "Liveness check (no token needed)",
	}, func(ctx context.Context, _ *struct{}) (*body[Health], error) {
		return &body[Health]{Body: Health{OK: true}}, nil
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-server", Method: http.MethodGet, Path: "/api/v1/server", Summary: "Server summary", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*body[*manager.ServerInfo], error) {
		info, err := m.Server()
		return &body[*manager.ServerInfo]{Body: info}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "list-protocols", Method: http.MethodGet, Path: "/api/v1/protocols", Summary: "List protocols", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*body[[]manager.ProtocolInfo], error) {
		list, err := m.Protocols()
		return &body[[]manager.ProtocolInfo]{Body: list}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "patch-protocol", Method: http.MethodPatch, Path: "/api/v1/protocols/{key}",
		Summary: "Enable, disable or change the port of a protocol", Security: bearer,
	}, func(ctx context.Context, in *struct {
		Key  string `path:"key" doc:"Protocol key, e.g. trojan"`
		Body manager.ProtocolPatch
	}) (*body[*manager.ProtocolInfo], error) {
		p, err := m.SetProtocol(in.Key, in.Body)
		return &body[*manager.ProtocolInfo]{Body: p}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "list-users", Method: http.MethodGet, Path: "/api/v1/users", Summary: "List users", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*body[[]manager.UserInfo], error) {
		list, err := m.Users()
		return &body[[]manager.UserInfo]{Body: list}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "create-user", Method: http.MethodPost, Path: "/api/v1/users", Summary: "Create a user",
		DefaultStatus: http.StatusCreated, Security: bearer,
	}, func(ctx context.Context, in *struct{ Body manager.NewUser }) (*body[*manager.UserInfo], error) {
		u, err := m.AddUser(in.Body)
		return &body[*manager.UserInfo]{Body: u}, toHTTP(err)
	})

	type userPath struct {
		Name string `path:"name" doc:"User name"`
	}

	huma.Register(a, huma.Operation{
		OperationID: "get-user", Method: http.MethodGet, Path: "/api/v1/users/{name}", Summary: "Get a user", Security: bearer,
	}, func(ctx context.Context, in *userPath) (*body[*manager.UserInfo], error) {
		u, err := m.User(in.Name)
		return &body[*manager.UserInfo]{Body: u}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "patch-user", Method: http.MethodPatch, Path: "/api/v1/users/{name}",
		Summary: "Enable, disable, set expiry, quota or note", Security: bearer,
	}, func(ctx context.Context, in *struct {
		Name string `path:"name" doc:"User name"`
		Body manager.UserPatch
	}) (*body[*manager.UserInfo], error) {
		u, err := m.UpdateUser(in.Name, in.Body)
		return &body[*manager.UserInfo]{Body: u}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "delete-user", Method: http.MethodDelete, Path: "/api/v1/users/{name}", Summary: "Delete a user",
		DefaultStatus: http.StatusNoContent, Security: bearer,
	}, func(ctx context.Context, in *userPath) (*struct{}, error) {
		return nil, toHTTP(m.DeleteUser(in.Name))
	})

	huma.Register(a, huma.Operation{
		OperationID: "reset-user", Method: http.MethodPost, Path: "/api/v1/users/{name}/reset",
		Summary: "Issue new credentials (old links stop working)", Security: bearer,
	}, func(ctx context.Context, in *userPath) (*body[*manager.UserInfo], error) {
		u, err := m.ResetUser(in.Name)
		return &body[*manager.UserInfo]{Body: u}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "reset-user-usage", Method: http.MethodPost, Path: "/api/v1/users/{name}/usage/reset",
		Summary: "Restart this month's traffic count and unblock the user (lifetime totals stay)", Security: bearer,
	}, func(ctx context.Context, in *userPath) (*body[*manager.UserInfo], error) {
		u, err := m.ResetUsage(in.Name)
		return &body[*manager.UserInfo]{Body: u}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-user-links", Method: http.MethodGet, Path: "/api/v1/users/{name}/links",
		Summary: "Share links and subscription URL", Security: bearer,
	}, func(ctx context.Context, in *userPath) (*body[*manager.LinkSet], error) {
		set, err := m.Links(in.Name)
		return &body[*manager.LinkSet]{Body: set}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-user-file", Method: http.MethodGet, Path: "/api/v1/users/{name}/files/{proto}",
		Summary: "The user's config file for a protocol (OpenVPN .ovpn, WireGuard .conf)", Security: bearer,
	}, func(ctx context.Context, in *struct {
		Name  string `path:"name" doc:"User name"`
		Proto string `path:"proto" doc:"Protocol key, e.g. wireguard or openvpn-udp"`
	}) (*struct {
		ContentType        string `header:"Content-Type"`
		ContentDisposition string `header:"Content-Disposition"`
		Body               []byte
	}, error) {
		filename, text, err := m.File(in.Name, in.Proto)
		if err != nil {
			return nil, toHTTP(err)
		}
		return &struct {
			ContentType        string `header:"Content-Type"`
			ContentDisposition string `header:"Content-Disposition"`
			Body               []byte
		}{ContentType: "application/octet-stream", ContentDisposition: `attachment; filename="` + filename + `"`, Body: []byte(text)}, nil
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-user-ovpn", Method: http.MethodGet, Path: "/api/v1/users/{name}/ovpn/{proto}",
		Summary: "The user's OpenVPN config file", Security: bearer,
	}, func(ctx context.Context, in *struct {
		Name  string `path:"name" doc:"User name"`
		Proto string `path:"proto" enum:"udp,tcp"`
	}) (*struct {
		ContentType string `header:"Content-Type"`
		Body        []byte
	}, error) {
		text, err := m.OVPN(in.Name, in.Proto)
		if err != nil {
			return nil, toHTTP(err)
		}
		return &struct {
			ContentType string `header:"Content-Type"`
			Body        []byte
		}{ContentType: "application/x-openvpn-profile", Body: []byte(text)}, nil
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-logs", Method: http.MethodGet, Path: "/api/v1/logs",
		Summary: "The last lines of the server log (oldest first)", Security: bearer,
	}, func(ctx context.Context, in *struct {
		Lines int `query:"lines" default:"200" minimum:"1" maximum:"5000" doc:"How many lines"`
	}) (*body[[]string], error) {
		lines, err := m.Logs(in.Lines)
		return &body[[]string]{Body: lines}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "get-certificate", Method: http.MethodGet, Path: "/api/v1/certificate",
		Summary: "Where the TLS certificate comes from (self-signed or ACME) and when it expires", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*body[*manager.CertInfo], error) {
		c, err := m.Certificate()
		return &body[*manager.CertInfo]{Body: c}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "enable-acme", Method: http.MethodPut, Path: "/api/v1/certificate/acme",
		Summary: "Get a real certificate for a domain (Let's Encrypt by default); the domain becomes the public address", Security: bearer,
	}, func(ctx context.Context, in *struct{ Body manager.ACMEPatch }) (*body[*manager.CertInfo], error) {
		c, err := m.EnableACME(in.Body)
		return &body[*manager.CertInfo]{Body: c}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "use-self-signed", Method: http.MethodDelete, Path: "/api/v1/certificate/acme",
		Summary: "Go back to the self-signed certificate", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*body[*manager.CertInfo], error) {
		c, err := m.UseSelfSigned()
		return &body[*manager.CertInfo]{Body: c}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "import-xray", Method: http.MethodPost, Path: "/api/v1/import/xray",
		Summary: "Import users (UUIDs, passwords, Reality identity) from an Xray/V2Ray server config", Security: bearer,
		MaxBodyBytes: 4 << 20,
	}, func(ctx context.Context, in *struct {
		Body struct {
			Config  string                    `json:"config" doc:"The server config.json as text" minLength:"2"`
			Options manager.ImportXrayOptions `json:"options" required:"false" doc:"host, name and adopt_reality; optional"`
		}
	}) (*body[*manager.ImportXrayResult], error) {
		res, err := m.ImportXray(in.Body.Config, in.Body.Options)
		return &body[*manager.ImportXrayResult]{Body: res}, toHTTP(err)
	})

	huma.Register(a, huma.Operation{
		OperationID: "download-backup", Method: http.MethodGet, Path: "/api/v1/backup",
		Summary: "Download a backup (.tar.gz) of the state, keys and certificates; it holds every secret", Security: bearer,
	}, func(ctx context.Context, _ *struct{}) (*struct {
		ContentType        string `header:"Content-Type"`
		ContentDisposition string `header:"Content-Disposition"`
		Body               []byte
	}, error) {
		var buf bytes.Buffer
		if err := m.Backup(&buf); err != nil {
			return nil, toHTTP(err)
		}
		return &struct {
			ContentType        string `header:"Content-Type"`
			ContentDisposition string `header:"Content-Disposition"`
			Body               []byte
		}{ContentType: "application/gzip", ContentDisposition: `attachment; filename="omnitun-server-backup.tar.gz"`, Body: buf.Bytes()}, nil
	})

	huma.Register(a, huma.Operation{
		OperationID: "reload", Method: http.MethodPost, Path: "/api/v1/reload",
		Summary: "Restart the VPN core now (it also reloads by itself within seconds)", Security: bearer,
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if o.Reload == nil {
			return nil, huma.Error501NotImplemented("the VPN core does not run in this process")
		}
		if err := o.Reload(); err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, nil
	})

	return router
}

// authenticate rejects requests to secured operations without the token.
func authenticate(a huma.API, current func() string) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if len(ctx.Operation().Security) == 0 {
			next(ctx)
			return
		}
		got, ok := strings.CutPrefix(ctx.Header("Authorization"), "Bearer ")
		token := ""
		if current != nil {
			token = current()
		}
		if !ok || token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			ctx.SetHeader("WWW-Authenticate", "Bearer")
			huma.WriteErr(a, ctx, http.StatusUnauthorized, "a valid bearer token is required")
			return
		}
		next(ctx)
	}
}
