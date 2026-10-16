package mobile

// Adapted from sing-box's experimental/libbox platform wrapper, with a
// smaller JSON-based native interface.

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"slices"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/common/logger"
	"github.com/sagernet/sing/common/x/list"

	"github.com/NoobforAl/omnitun/core/singbox"
)

var errUnsupported = errors.New("not supported on this platform")

// tunConfig is sent to Platform.OpenTun.
type tunConfig struct {
	MTU               uint32       `json:"mtu"`
	Inet4Address      []string     `json:"inet4_address"`
	Inet6Address      []string     `json:"inet6_address"`
	DNSServers        []string     `json:"dns_servers"`
	AutoRoute         bool         `json:"auto_route"`
	Inet4RouteRange   []string     `json:"inet4_route_range"`
	Inet6RouteRange   []string     `json:"inet6_route_range"`
	Inet4RouteExclude []string     `json:"inet4_route_exclude"`
	Inet6RouteExclude []string     `json:"inet6_route_exclude"`
	IncludePackage    []string     `json:"include_package"`
	ExcludePackage    []string     `json:"exclude_package"`
	ExcludeSelf       bool         `json:"exclude_self,omitempty"`
	HTTPProxy         *httpProxyCf `json:"http_proxy,omitempty"`
}

type httpProxyCf struct {
	Server       string   `json:"server"`
	Port         uint16   `json:"port"`
	BypassDomain []string `json:"bypass_domain"`
}

// netInterface is returned by Platform.Interfaces.
type netInterface struct {
	Name      string   `json:"name"`
	Index     int      `json:"index"`
	MTU       int      `json:"mtu"`
	Addresses []string `json:"addresses"` // CIDR
	Flags     uint32   `json:"flags"`     // unix IFF_* bits
	Type      string   `json:"type"`      // wifi, cellular, ethernet, other
	DNS       []string `json:"dns"`
	Metered   bool     `json:"metered"`
}

func prefixStrings(p []netip.Prefix, keep func(netip.Prefix) bool) []string {
	out := []string{}
	for _, v := range p {
		if keep == nil || keep(v) {
			out = append(out, v.String())
		}
	}
	return out
}

type platformWrapper struct {
	p              Platform
	networkManager adapter.NetworkManager
	monitor        *defaultMonitor

	mu         sync.Mutex
	myTunAddrs []netip.Addr
}

var _ adapter.PlatformInterface = (*platformWrapper)(nil)

func newPlatformWrapper(p Platform) *platformWrapper { return &platformWrapper{p: p} }

func (w *platformWrapper) Initialize(networkManager adapter.NetworkManager) error {
	w.networkManager = networkManager
	return nil
}

func (w *platformWrapper) UsePlatformAutoDetectInterfaceControl() bool { return true }

func (w *platformWrapper) AutoDetectInterfaceControl(fd int) error { return w.p.Protect(int32(fd)) }

func (w *platformWrapper) UsePlatformInterface() bool { return true }

func (w *platformWrapper) OpenInterface(options *tun.Options, platformOptions option.TunPlatformOptions) (tun.Tun, error) {
	if len(options.IncludeUID) > 0 || len(options.ExcludeUID) > 0 || len(options.IncludeAndroidUser) > 0 {
		return nil, errors.New("platform: uid and android_user options are not supported")
	}
	ranges, err := options.BuildAutoRouteRanges(true)
	if err != nil {
		return nil, err
	}
	dns, err := options.DNSServerAddress()
	if err != nil {
		return nil, err
	}
	cfg := tunConfig{
		MTU:               options.MTU,
		Inet4Address:      prefixStrings(options.Inet4Address, nil),
		Inet6Address:      prefixStrings(options.Inet6Address, nil),
		AutoRoute:         options.AutoRoute,
		Inet4RouteRange:   prefixStrings(ranges, func(p netip.Prefix) bool { return p.Addr().Is4() }),
		Inet6RouteRange:   prefixStrings(ranges, func(p netip.Prefix) bool { return p.Addr().Is6() }),
		Inet4RouteExclude: prefixStrings(options.Inet4RouteExcludeAddress, nil),
		Inet6RouteExclude: prefixStrings(options.Inet6RouteExcludeAddress, nil),
		IncludePackage:    append([]string{}, options.IncludePackage...),
		ExcludePackage:    append([]string{}, options.ExcludePackage...),
	}
	if i := slices.Index(cfg.ExcludePackage, singbox.SelfPackage); i >= 0 {
		cfg.ExcludePackage = slices.Delete(cfg.ExcludePackage, i, i+1)
		cfg.ExcludeSelf = true
	}
	for _, a := range dns {
		cfg.DNSServers = append(cfg.DNSServers, a.String())
	}
	if hp := platformOptions.HTTPProxy; hp != nil && hp.Enabled {
		cfg.HTTPProxy = &httpProxyCf{Server: hp.Server, Port: hp.ServerPort, BypassDomain: hp.BypassDomain}
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	fd, err := w.p.OpenTun(string(raw))
	if err != nil {
		return nil, err
	}
	options.Name, err = tunName(fd)
	if err != nil {
		return nil, err
	}
	options.InterfaceMonitor.RegisterMyInterface(options.Name)
	dupFd, err := dupFD(int(fd))
	if err != nil {
		return nil, err
	}
	options.FileDescriptor = dupFd

	addrs := make([]netip.Addr, 0, len(options.Inet4Address)+len(options.Inet6Address))
	for _, p := range append(append([]netip.Prefix{}, options.Inet4Address...), options.Inet6Address...) {
		addrs = append(addrs, p.Addr())
	}
	w.mu.Lock()
	w.myTunAddrs = addrs
	w.mu.Unlock()
	return tun.New(*options)
}

func (w *platformWrapper) ProcessPlatformOptions(option.TunPlatformOptions) error { return nil }

func (w *platformWrapper) MyInterfaceAddress() []netip.Addr {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.myTunAddrs
}

func (w *platformWrapper) UsePlatformDefaultInterfaceMonitor() bool { return true }

func (w *platformWrapper) CreateDefaultInterfaceMonitor(l logger.Logger) tun.DefaultInterfaceMonitor {
	w.monitor = &defaultMonitor{w: w, logger: l}
	return w.monitor
}

func (w *platformWrapper) UsePlatformNetworkInterfaces() bool { return true }

func (w *platformWrapper) NetworkInterfaces() ([]adapter.NetworkInterface, error) {
	raw, err := w.p.Interfaces()
	if err != nil {
		return nil, err
	}
	var list []netInterface
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil, err
	}
	var defaultIndex = -1
	var expensive, constrained bool
	if m := w.monitor; m != nil {
		m.mu.Lock()
		if m.defaultInterface != nil {
			defaultIndex = m.defaultInterface.Index
		}
		expensive, constrained = m.expensive, m.constrained
		m.mu.Unlock()
	}
	seen := map[string]bool{}
	var out []adapter.NetworkInterface
	for _, ni := range list {
		if seen[ni.Name] {
			continue
		}
		seen[ni.Name] = true
		var addrs []netip.Prefix
		for _, a := range ni.Addresses {
			if p, err := netip.ParsePrefix(a); err == nil {
				addrs = append(addrs, p)
			}
		}
		typ, ok := C.StringToInterfaceType[ni.Type]
		if !ok {
			typ = C.InterfaceTypeOther
		}
		isDefault := ni.Index == defaultIndex
		out = append(out, adapter.NetworkInterface{
			Interface: control.Interface{
				Index:     ni.Index,
				MTU:       ni.MTU,
				Name:      ni.Name,
				Addresses: addrs,
				Flags:     linkFlags(ni.Flags),
			},
			Type:        typ,
			DNSServers:  ni.DNS,
			Expensive:   ni.Metered || isDefault && expensive,
			Constrained: isDefault && constrained,
		})
	}
	return out, nil
}

// Unused platform features.

func (w *platformWrapper) UnderNetworkExtension() bool              { return false }
func (w *platformWrapper) NetworkExtensionIncludeAllNetworks() bool { return false }
func (w *platformWrapper) ClearDNSCache()                           {}
func (w *platformWrapper) RequestPermissionForWIFIState() error     { return nil }
func (w *platformWrapper) ReadWIFIState(context.Context) adapter.WIFIState {
	return adapter.WIFIState{}
}
func (w *platformWrapper) UsePlatformConnectionOwnerFinder() bool { return false }
func (w *platformWrapper) FindConnectionOwner(*adapter.FindConnectionOwnerRequest) (*adapter.ConnectionOwner, error) {
	return nil, errUnsupported
}
func (w *platformWrapper) UsePlatformWIFIMonitor() bool                 { return false }
func (w *platformWrapper) UsePlatformNotification() bool                { return false }
func (w *platformWrapper) SendNotification(*adapter.Notification) error { return nil }
func (w *platformWrapper) CancelNotification(string, int32) error       { return nil }
func (w *platformWrapper) UsePlatformNeighborResolver() bool            { return false }
func (w *platformWrapper) StartNeighborMonitor(adapter.NeighborUpdateListener) error {
	return errUnsupported
}
func (w *platformWrapper) CloseNeighborMonitor(adapter.NeighborUpdateListener) error { return nil }
func (w *platformWrapper) UsePlatformShell() bool                                    { return false }
func (w *platformWrapper) CheckPlatformShell() error                                 { return errUnsupported }
func (w *platformWrapper) OpenShellSession(*adapter.PlatformUser, string, []string, string, int32, int32) (adapter.ShellSession, error) {
	return nil, errUnsupported
}
func (w *platformWrapper) LookupUser(string) (*adapter.PlatformUser, error) {
	return nil, errUnsupported
}
func (w *platformWrapper) LookupSFTPServer() (string, error)     { return "", errUnsupported }
func (w *platformWrapper) ReadSystemSSHHostKey() ([]byte, error) { return nil, errUnsupported }
func (w *platformWrapper) TailscaleHostname() string             { return "" }
func (w *platformWrapper) UsePlatformBridge() bool               { return false }
func (w *platformWrapper) CreateBridge(adapter.BridgeOptions) (adapter.BridgeSession, error) {
	return nil, errUnsupported
}

// defaultMonitor tracks the default network reported by the platform.
type defaultMonitor struct {
	w      *platformWrapper
	logger logger.Logger

	mu               sync.Mutex
	callbacks        list.List[tun.DefaultInterfaceUpdateCallback]
	myInterfaces     []string
	defaultInterface *control.Interface
	initialized      bool
	expensive        bool
	constrained      bool
}

var (
	_ tun.DefaultInterfaceMonitor = (*defaultMonitor)(nil)
	_ DefaultNetworkListener      = (*defaultMonitor)(nil)
)

func (m *defaultMonitor) Start() error { return m.w.p.StartMonitor(m) }
func (m *defaultMonitor) Close() error { return m.w.p.CloseMonitor() }

func (m *defaultMonitor) DefaultInterface() *control.Interface {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.defaultInterface
}

func (m *defaultMonitor) OverrideAndroidVPN() bool { return false }
func (m *defaultMonitor) AndroidVPNEnabled() bool  { return false }

func (m *defaultMonitor) RegisterCallback(cb tun.DefaultInterfaceUpdateCallback) *list.Element[tun.DefaultInterfaceUpdateCallback] {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callbacks.PushBack(cb)
}

func (m *defaultMonitor) UnregisterCallback(e *list.Element[tun.DefaultInterfaceUpdateCallback]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks.Remove(e)
}

func (m *defaultMonitor) RegisterMyInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.myInterfaces = append(m.myInterfaces, name)
}

func (m *defaultMonitor) MyInterfaces() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.myInterfaces
}

// UpdateDefaultInterface is called by the platform. It runs on a fresh
// goroutine because platform callback threads have small stacks.
func (m *defaultMonitor) UpdateDefaultInterface(name string, index int32, expensive, constrained bool) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		m.update(name, index, expensive, constrained)
	}()
	<-done
}

func (m *defaultMonitor) update(name string, index int32, expensive, constrained bool) {
	m.mu.Lock()
	m.expensive, m.constrained = expensive, constrained
	m.mu.Unlock()
	if nm := m.w.networkManager; nm != nil {
		if err := nm.UpdateInterfaces(); err != nil {
			m.logger.Error("update interfaces: ", err)
		}
	}
	m.mu.Lock()
	if index == -1 {
		m.defaultInterface = nil
		m.initialized = true
		cbs := m.callbacks.Array()
		m.mu.Unlock()
		for _, cb := range cbs {
			cb(nil, 0)
		}
		return
	}
	var next *control.Interface
	if nm := m.w.networkManager; nm != nil {
		var err error
		next, err = nm.InterfaceFinder().ByIndex(int(index))
		if err != nil {
			m.mu.Unlock()
			m.logger.Error("find default interface ", name, ": ", err)
			return
		}
	} else {
		next = &control.Interface{Name: name, Index: int(index)}
	}
	old := m.defaultInterface
	m.defaultInterface = next
	if m.initialized && old != nil && old.Name == next.Name && old.Index == next.Index {
		m.mu.Unlock()
		return
	}
	m.initialized = true
	cbs := m.callbacks.Array()
	m.mu.Unlock()
	for _, cb := range cbs {
		cb(next, 0)
	}
}

// linkFlags converts unix IFF_* bits (sent by the platform) to net.Flags.
func linkFlags(raw uint32) net.Flags {
	const (
		iffUp           = 0x1
		iffBroadcast    = 0x2
		iffLoopback     = 0x8
		iffPointToPoint = 0x10
		iffRunning      = 0x40
		iffMulticast    = 0x1000
	)
	var f net.Flags
	if raw&iffUp != 0 {
		f |= net.FlagUp
	}
	if raw&iffRunning != 0 {
		f |= net.FlagRunning
	}
	if raw&iffBroadcast != 0 {
		f |= net.FlagBroadcast
	}
	if raw&iffLoopback != 0 {
		f |= net.FlagLoopback
	}
	if raw&iffPointToPoint != 0 {
		f |= net.FlagPointToPoint
	}
	if raw&iffMulticast != 0 {
		f |= net.FlagMulticast
	}
	return f
}
