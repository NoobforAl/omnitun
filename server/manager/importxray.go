package manager

import (
	"fmt"

	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/v2ray"
)

// ImportXrayOptions controls moving an Xray/V2Ray server to Omnitun.
type ImportXrayOptions struct {
	// Host is the public IP or domain; needed when no server exists yet.
	Host string `json:"host,omitempty"`
	// Name is the server name for a newly created server.
	Name string `json:"name,omitempty"`
	// AdoptReality replaces an existing server's Reality identity with the
	// imported one (a new server always adopts it). Existing Reality links
	// then keep working, but links issued by this server before stop.
	AdoptReality bool `json:"adopt_reality,omitempty"`
}

// ImportXrayResult says what the import did.
type ImportXrayResult struct {
	CreatedServer  bool     `json:"created_server"`
	AdoptedReality bool     `json:"adopted_reality"`
	UsersAdded     []string `json:"users_added"`
	UsersSkipped   []string `json:"users_skipped,omitempty" doc:"Names that already exist"`
	Notes          []string `json:"notes,omitempty"`
}

// ImportXray reads an Xray/V2Ray server config and brings its users (with
// their UUIDs and passwords) and its Reality identity into this server,
// creating the server first if needed.
func (m *Manager) ImportXray(text string, o ImportXrayOptions) (*ImportXrayResult, error) {
	cfg, err := v2ray.ParseServer(text)
	if err != nil {
		return nil, invalid("%v", err)
	}
	res := &ImportXrayResult{Notes: cfg.Notes}
	if !m.Store.Exists() {
		opts := InitOptions{PublicHost: o.Host, Name: o.Name}
		if cfg.Reality != nil {
			opts.RealityHandshake = cfg.Reality.Handshake
		}
		if _, err := m.Init(opts); err != nil {
			return nil, err
		}
		res.CreatedServer = true
		o.AdoptReality = true
	}
	err = m.Store.Update(func(st *store.State) error {
		if cfg.Reality != nil && o.AdoptReality {
			r := &st.Server.Reality
			r.PrivateKey, r.PublicKey = cfg.Reality.PrivateKey, cfg.Reality.PublicKey
			if cfg.Reality.ShortID != "" {
				r.ShortID = cfg.Reality.ShortID
			}
			if cfg.Reality.Handshake != "" {
				r.Handshake = cfg.Reality.Handshake
			}
			if cfg.Reality.Port > 0 {
				st.Server.Protocols[store.VLESSReality].Port = cfg.Reality.Port
			}
			res.AdoptedReality = true
		}
		for _, iu := range cfg.Users {
			if st.User(iu.Name) != nil {
				res.UsersSkipped = append(res.UsersSkipped, iu.Name)
				continue
			}
			u, err := store.AddUser(st, iu.Name)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrInvalid, err)
			}
			if iu.UUID != "" {
				u.UUID = iu.UUID
			}
			if iu.Password != "" {
				u.Password = iu.Password
			}
			u.Note = iu.Note
			res.UsersAdded = append(res.UsersAdded, u.Name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if res.UsersAdded == nil {
		res.UsersAdded = []string{}
	}
	return res, nil
}
