// Package mobile is the gomobile-bound entry point for Android (and later iOS).
//
// It runs the same control server as omnitun-svc inside the app process, so
// the Flutter UI talks to it exactly like on desktop, and plugs the OS VPN
// API into sing-box through Platform.
package mobile

import (
	"encoding/json"
	"sync"

	"github.com/NoobforAl/omnitun/core/control"
	"github.com/NoobforAl/omnitun/core/engine"
)

// Platform is implemented natively (Kotlin VpnService on Android).
// gomobile restricts signatures to basic types, so structured data is JSON.
type Platform interface {
	// OpenTun establishes the VPN interface described by tunJSON (see tunConfig)
	// and returns its file descriptor. The caller keeps ownership of the fd.
	OpenTun(tunJSON string) (int32, error)
	// Protect makes a socket bypass the VPN (VpnService.protect).
	Protect(fd int32) error
	// Interfaces returns the device's network interfaces as JSON (see netInterface).
	Interfaces() (string, error)
	// StartMonitor begins reporting default-network changes to listener.
	StartMonitor(listener DefaultNetworkListener) error
	// CloseMonitor stops reporting.
	CloseMonitor() error
	// StateChanged reports the engine state as JSON (see notifyState): status is
	// "connecting", "connected", "disconnected" or "error"; it is called about once
	// a second while connected, with traffic, so the notification can show speed.
	// The native side stops its VPN service on "disconnected" and "error".
	StateChanged(stateJSON string)
	// StartPlatformVPN starts a tunnel the OS implements (IKEv2) for the
	// profile JSON and returns at once; report progress with PlatformVPNStatus.
	StartPlatformVPN(profileJSON string) error
	// StopPlatformVPN stops it. It must not call PlatformVPNStatus synchronously.
	StopPlatformVPN()
}

// PlatformVPNStatus reports the OS tunnel state: "connected", "disconnected"
// or "failed" (with a message).
func PlatformVPNStatus(status, message string) {
	mu.Lock()
	s := srv
	mu.Unlock()
	if s != nil {
		s.Engine().PlatformVPNStatus(status, message)
	}
}

// DefaultNetworkListener receives default-network updates from the platform.
// index is -1 when there is no network.
type DefaultNetworkListener interface {
	UpdateDefaultInterface(name string, index int32, expensive bool, constrained bool)
}

var (
	mu   sync.Mutex
	srv  *control.Server
	info string
)

// Start launches the in-process control server (idempotent) and returns
// {"addr":"127.0.0.1:port","token":"..."} for the UI to connect to.
func Start(dataDir string, platform Platform) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	if srv != nil {
		return info, nil
	}
	s, err := control.NewServer(control.ServerConfig{
		DataDir: dataDir,
		// Android pauses apps in the background, so a quiet UI connection means
		// nothing; the VPN service stops the tunnel when the task is removed.
		Grace:       -1,
		Platform:    newPlatformWrapper(platform),
		Tun:         tunProvider{platform},
		PlatformVPN: platform,
		// Go cannot read the Android system CA store; use the embedded Mozilla bundle.
		CertificateStore: "mozilla",
		OnState: func(st engine.State) {
			b, _ := json.Marshal(notifyState{
				Status: string(st.Status), Profile: st.ProfileName, UpBps: st.UpBps, DownBps: st.DownBps,
			})
			platform.StateChanged(string(b))
		},
	})
	if err != nil {
		return "", err
	}
	addr, err := s.Listen("127.0.0.1:0")
	if err != nil {
		return "", err
	}
	go s.Serve()
	b, _ := json.Marshal(map[string]string{"addr": addr.String(), "token": s.Token()})
	srv, info = s, string(b)
	return info, nil
}

// StopTunnel stops the running tunnel but keeps the server (e.g. VPN revoked by the OS).
func StopTunnel() {
	mu.Lock()
	s := srv
	mu.Unlock()
	if s != nil {
		s.Engine().Stop()
	}
}

// Stop shuts everything down.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if srv != nil {
		srv.Close()
		srv, info = nil, ""
	}
}

// notifyState is the JSON passed to Platform.StateChanged.
type notifyState struct {
	Status  string `json:"status"`
	Profile string `json:"profile,omitempty"`
	UpBps   int64  `json:"up_bps"`
	DownBps int64  `json:"down_bps"`
}

// tunProvider opens the VPN interface for engines other than sing-box (AmneziaWG).
type tunProvider struct{ p Platform }

func (t tunProvider) OpenTun(req engine.TunRequest) (int, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return -1, err
	}
	fd, err := t.p.OpenTun(string(raw))
	return int(fd), err
}

func (t tunProvider) Protect(fd int) error { return t.p.Protect(int32(fd)) }
