# Omnitun — Project Plan

> Cross-platform VPN **client** (GUI) and easy-to-run VPN **server** (CLI + TUI).
> License: **MIT**. First-class targets: **Windows + Android**. iOS / macOS / Linux: build configs and stubs only, implemented later.

---

## 0. Original requirements (summary)

- Client app (GUI) connects to any supported VPN/proxy protocol; server side is a CLI with a simple terminal UI for user and config management.
- Protocols: WireGuard, OpenVPN (UDP/TCP), IKEv2/IPsec, SSTP, L2TP/IPsec, Lightway, NordLynx, VLESS+Reality, Shadowsocks (AEAD), Trojan, Hysteria2, TUIC, NaiveProxy, AmneziaWG, obfs4, Snowflake, WebTunnel, Obfuscated OpenVPN (XOR/Stunnel), Stealth (WireGuard over TLS), Tor (Rust-based), plus plain proxy config.
- Import any standard config file or share link; config groups; subscriptions from a URL (optionally authenticated).
- Auto-switch to another config when a protocol fails; show a distinct "no internet" error when the network itself is down.
- Request OS VPN permission when needed; when the user closes the app or the process is killed, the tunnel and background jobs must stop.
- Languages: Go (most protocols, CLI), Rust (Tor, Rust-native protocols), C/C++ only where nothing else exists.

---

## 1. Key decisions

### 1.1 UI framework: **Flutter — keep it**

Flutter is the right choice for this project. Compared with the alternatives:

| Option | Verdict |
|---|---|
| **Flutter** ✅ | One codebase for Android/Windows/iOS/macOS/Linux; mature FFI (`dart:ffi`) to call Go/Rust libraries; proven for exactly this kind of app (Hiddify, many v2ray/WireGuard clients). Weak spot: Windows tunnel code is native work anyway (we do it in Go, not Flutter). |
| Compose Multiplatform | Great on Android, but Windows desktop (tray, elevation, installers) is less mature and iOS is newer. Good only if the team is Kotlin-first. |
| Tauri v2 (Rust + web UI) | Excellent on desktop and fits Rust; mobile is younger, and Android `VpnService` still needs Kotlin plugin code. Reasonable second choice. |
| React Native | Windows support is the weakest of the four. |

The UI is a thin layer: **all networking lives in the Go/Rust core**, so the UI framework can be swapped later without touching protocols.

### 1.2 Engine and dependencies

Decision (2026-10-15): dependencies are chosen on technical merit; license compatibility is not a constraint. The **main proxy engine is sing-box** (embedded as a Go library, pinned `v1.14.2`). It already provides VLESS/Reality, VMess, Trojan, Shadowsocks, Hysteria2, TUIC, WireGuard, the TUN inbound (gVisor/system stack), DNS, routing and urltest/selector groups, which removes most of the protocol work.

| Component | Role |
|---|---|
| **sing-box** (Go lib) | VLESS/Reality, VMess, Trojan, SS, Hysteria2, TUIC, WireGuard/NordLynx, TUN, DNS, routing, groups |
| amneziawg-go | AmneziaWG (not supported by sing-box) |
| Arti (Rust) + lyrebird/snowflake/webtunnel | Tor with bridges |
| NaiveProxy | Separate binary (Chromium network stack) |
| OpenVPN | sing-box 1.14 has an `openvpn` outbound (`with_openvpn` tag) — try it first; fall back to OpenVPN 2.6 binary (needed for the XOR patch) |
| OpenConnect / NaiveProxy | Also built into sing-box 1.14 (`with_openconnect`, `with_naive_outbound` — naive needs cronet, so it's off for now) |
| Lightway (Rust, Linux/macOS only today) | Separate binary, low priority |
| Windows RAS / Android `Ikev2VpnProfile` | IKEv2, L2TP, SSTP (OS-native) |
| strongSwan / xl2tpd / accel-ppp (server) | Installed as system packages by the server CLI |

IKEv2, L2TP/IPsec and SSTP on **Windows** use the built-in RAS VPN client (no third-party code). On **Android**: IKEv2 uses the platform `Ikev2VpnProfile` (Android 11+); SSTP uses a Kotlin port of Open SSTP Client (MIT — verify before use); **L2TP is not available** (Android 12 removed it) — the UI shows "not supported on this device".

### 1.3 Language split

| Language | Responsibility |
|---|---|
| **Go** | `core` engine (tunnel manager, protocols, tun2socks, routing, health checks, failover, config parser, subscriptions); server CLI/TUI; Windows privileged service |
| **Rust** | `tor` module (Arti), exposed through a C ABI; later Rust-native protocols |
| **Kotlin** | Android `VpnService`, foreground service, IKEv2/SSTP platform glue |
| **Dart/Flutter** | UI only |
| C/C++ | Only as glue if a protocol has no Go/Rust option |

---

## 2. Architecture

```
┌──────────────────────────── Flutter UI (Dart) ─────────────────────────────┐
│ screens · state (Riverpod) · local DB (drift/SQLite) · i18n · theming       │
└──────────────┬─────────────────────────────────────────┬────────────────────┘
        Android│ MethodChannel + FFI              Desktop│ loopback TCP, JSON lines + token
┌──────────────▼──────────────┐          ┌───────────────▼─────────────────────┐
│ Kotlin VpnService (fg svc)  │          │ omnitun-svc.exe (Windows Service,   │
│  - prepare() permission     │          │  runs as SYSTEM; UI runs unelevated) │
│  - builds TUN, passes fd    │          │  - Wintun adapter, routes, DNS, WFP  │
│  - onRevoke / onDestroy     │          │    kill-switch                       │
└──────────────┬──────────────┘          └───────────────┬─────────────────────┘
               └──────────────► libcore (Go) ◄───────────┘
                 .aar via gomobile │ .dll / exe on Windows
   ┌──────────────────────────────────────────────┬───────────────┬────────────┐
   │ sing-box: VLESS/Reality, VMess, Trojan, SS,  │ amneziawg-go  │ Failover & │
   │ Hysteria2, TUIC, WireGuard/NordLynx, TUN, DNS│ (AWG)         │ health     │
   └──────────────────────────────────────────────┴───────────────┴────────────┘
   │ PT: obfs4 · snowflake · webtunnel ──► libtor (Rust, Arti) via C ABI       │
   │ External add-ons (separate processes): openvpn, naive, lightway, stunnel  │
   │ OS-native: Windows RAS (IKEv2/L2TP/SSTP), Android Ikev2VpnProfile         │
```

### 2.1 Core engine API (one API for every platform)

Small, versioned, JSON-in/JSON-out C ABI (`core_start(json)`, `core_stop()`, `core_status()`, `core_subscribe_events(cb)`, `core_test(profile_id)`, `core_import(text)`). Events: `state_changed`, `traffic`, `log`, `failover`, `error{code}`. The same API is used by the Android service, the Windows service and the server CLI.

### 2.2 Tunnel modes

1. **Native L3 tunnels** — WireGuard, AmneziaWG, NordLynx, OpenVPN: packets go straight from the TUN device.
2. **Proxy protocols** — VLESS, Trojan, SS, Hysteria2, TUIC, Naive, Tor: sing-box TUN inbound (gVisor/system stack) → outbound. DNS is handled in-core (fake-IP or DoH) to avoid leaks.
3. **OS-native** — IKEv2, L2TP, SSTP on Windows: the core drives RAS and only monitors status.
4. **Proxy-only mode** (no VPN permission needed) — local mixed SOCKS5/HTTP port + optional Windows system proxy. This also covers "Proxy config" from the requirements.

### 2.3 Chained / obfuscated protocols

- **Stealth (WireGuard over TLS)**: WireGuard UDP → local Go UDP-over-TLS/WebSocket wrapper (wstunnel-compatible) → server.
- **Obfuscated OpenVPN**: XOR via the OpenVPN add-on built with the XOR patch; Stunnel mode via our own Go TLS wrapper (no stunnel dependency).
- **Tor bridges**: Arti with obfs4/snowflake/webtunnel transports compiled into the Go core and launched as managed PTs.
- **NordLynx**: WireGuard plus NordVPN's API — the user enters an access token, we fetch the WireGuard key and server list.

---

## 3. Features

### 3.1 Config import and formats

| Input | Examples |
|---|---|
| Files | `.conf` (WireGuard/AmneziaWG), `.ovpn`, Xray/V2Ray JSON, sing-box JSON (parsed only, not executed), Clash/Mihomo YAML, SIP008 JSON, Tor `torrc` bridge lines |
| Share links | `vless://`, `vmess://`, `trojan://`, `ss://`, `hysteria2://` / `hy2://`, `tuic://`, `naive+https://`, `wireguard://`, `awg://`, `bridge` lines |
| Input methods | file picker, clipboard paste, QR scan (camera / image), drag-and-drop (desktop), deep link `omnitun://import?url=` |

Everything is normalized into one internal **Profile** model (`protocol`, `server`, `auth`, `transport`, `tls`, `obfs`, `routing`, `meta`). Exporting converts back to share links / QR.

### 3.2 Groups and subscriptions

- **Group** = an ordered list of profiles plus a strategy: `manual`, `priority (failover)`, `lowest-latency`, `round-robin`.
- **Subscription** = a group sourced from a URL, with auto-refresh interval, last-update time, traffic/expiry from the `subscription-userinfo` header, and `profile-update-interval`.
- **Subscription auth**: none, HTTP Basic, Bearer token, custom headers, token in query string, client certificate (mTLS), custom User-Agent. Secrets are stored in the OS keystore (Android Keystore / Windows DPAPI) via `flutter_secure_storage`.

### 3.3 Auto-switch and error handling

1. Before connecting, check the underlying network (OS connectivity API, then a direct DNS + HTTP probe).
   - Offline → **"No internet connection"** error, do **not** fail over, retry automatically when the network returns.
2. Connect to the selected profile; health check by HTTP probe through the tunnel (`generate_204`) with a timeout.
3. On failure: handshake timeout, auth error, probe failure ×N, or the tunnel dropping → mark the profile unhealthy (with backoff) and try the next one in the group by strategy. The UI shows a toast: "Switched to X (Y failed: reason)".
4. Periodic background probes while connected; network change (Wi-Fi ↔ mobile) triggers a reconnect, not a failover.
5. Error codes are typed (`NO_NETWORK`, `DNS_FAIL`, `HANDSHAKE_TIMEOUT`, `AUTH_FAILED`, `TLS_ERROR`, `BLOCKED_SUSPECTED`, `PERMISSION_DENIED`, `UNSUPPORTED_ON_PLATFORM`), and each has a human-readable, translatable message.

### 3.4 Permissions and lifecycle

**Android**
- `VpnService.prepare()` → system consent dialog; `POST_NOTIFICATIONS` on Android 13+; foreground service type `systemExempted` / `specialUse` as required by the current target SDK.
- Stop the tunnel on: user disconnect, `onRevoke()` (another VPN took over), notification "Disconnect" action, and app removed from recents if "Stop VPN when app is closed" is on (default **on**, per requirements).
- Process killed: the TUN fd dies with the process, so Android tears the VPN down automatically. Optional: support Always-on VPN and "Block connections without VPN" from system settings.

**Windows**
- UI runs unelevated. `omnitun-svc` is installed once by the MSI/MSIX installer (UAC prompt at install time only).
- Safety: the service holds a **session lease** — the UI keeps a token-authenticated loopback control connection (`core/control`, newline-delimited JSON) open and pings every 4 s. If the connection closes or pings stop (UI closed or killed), the service tears down the tunnel after a 5 s grace period. In dev/unprivileged mode the UI launches `omnitun-svc --parent-pid`, which exits when the UI process dies (verified).
- All add-on processes (openvpn, naive, …) are started inside a **Job Object** with `KILL_ON_JOB_CLOSE`, so they can never outlive the service.
- On service start, stale Wintun adapters, routes and firewall rules from a crash are cleaned up.
- Optional kill switch through WFP while connected.

### 3.5 Routing and extras (after MVP)

Split tunneling (per-app on Android, per-route on Windows), bypass LAN, geoip/geosite rules, custom DNS (DoH/DoT), IPv6 on/off, MTU, latency test for all profiles, traffic statistics, logs view with export.

---

## 4. Server CLI (`omnitun-server`, Go, Linux first)

- Commands: `init`, `install <protocol>`, `user add|del|list|disable|quota|expire`, `config export <user> [--qr|--link|--file]`, `sub serve` (subscription HTTP endpoint with per-user token), `status`, `logs`, `uninstall`.
- **TUI** (`omnitun-server tui`, Bubble Tea): user table, per-user configs and QR, protocol status, traffic, expiry.
- Storage: SQLite (`modernc.org/sqlite`, pure Go).
- In-process protocols: WireGuard (kernel module via `wgctrl`, or wireguard-go), AmneziaWG, sing-box server (VLESS+Reality, Trojan, SS, Hysteria2, TUIC), Stealth-WG wrapper, Tor bridge (obfs4/webtunnel).
- Orchestrated system packages (installed via apt/dnf, configured by templates): OpenVPN, strongSwan (IKEv2), xl2tpd (L2TP), accel-ppp or SoftEther (SSTP), Caddy + forwardproxy (NaiveProxy), Lightway server (add-on).
- Handles firewall/NAT (nftables), sysctl `ip_forward`, TLS certificates (ACME via certmagic) and systemd units.
- Linked by the client: the server can output a single subscription URL with all of a user's protocols, ordered for failover.

---

## 5. UI / UX design

### 5.1 Principles (from research on current VPN apps, e.g. NordVPN's 2026 redesigns)

1. **One-tap connect** — a large, central connect button; the home screen is mostly that button.
2. **Honest status** — never show "Connected" unless the health probe passed. Distinct states: Disconnected (neutral), Connecting (animated), Connected (accent color + timer), Error (red + reason + action), No internet (grey + wifi-off icon).
3. **Simple for beginners, deep for experts** — "Recommended / Auto" profile by default; protocol, transport and routing details live in an Advanced section.
4. **Show why** — when failover happens or a protocol fails, say what happened and what to do.
5. **Don't use an ✕ that looks like "turn off VPN"** — use back navigation.
6. **RTL and localization from day one** (English, Persian, Russian, Chinese, Arabic…).

### 5.2 Navigation (adaptive)

- **Phone**: bottom `NavigationBar` — Home · Profiles · Logs · Settings.
- **Desktop/tablet (≥ 840 px)**: `NavigationRail` on the left with a two-pane layout (list + details).
- **Windows**: system tray icon (connect/disconnect, profile quick-switch, quit); close-to-tray is configurable.

### 5.3 Screens

| Screen | Content |
|---|---|
| Home | Big connect button, current profile chip (tap to change), status text + timer, ↑/↓ speed, IP/location after connect, failover toasts |
| Profiles | Groups as sections; each row: flag, name, protocol badge, latency (colored), health dot; ⋮ menu: edit, share/QR, test, delete; FAB "+" → Import (file / clipboard / QR / URL / manual) |
| Subscription editor | URL, auth type selector (none/basic/bearer/headers/mTLS), refresh interval, usage/expiry bar |
| Profile editor | Protocol-specific form generated from a schema; Advanced section collapsed |
| Logs | Filterable live log with copy/export |
| Settings | Language, theme (system/light/dark), mode (VPN/proxy-only), stop on close, kill switch, DNS, split tunneling, start on boot, auto-connect |
| Onboarding | 2–3 screens: what it does → import first config → VPN permission explanation **before** the OS dialog |

### 5.4 Visual system

- Material 3 with a custom seed color; dark mode by default on desktop, system on mobile.
- Semantic colors: connected = green/teal, connecting = amber, error = red, offline = grey.
- Protocol badges with fixed short labels (WG, AWG, VLESS, TRJ, SS, HY2, TUIC, OVPN, IKEv2, SSTP, TOR).
- Motion: one pulse animation while connecting; nothing else animates continuously.
- Accessibility: 48 dp touch targets, state never shown by color alone (icon + text), large-font support.

### 5.5 Flutter packages

`flutter_riverpod` (state), `go_router` (navigation), `drift` (SQLite), `freezed` + `json_serializable` (models), `flutter_secure_storage` (secrets), `slang` or `intl` (i18n, RTL), `mobile_scanner` (QR), `qr_flutter` (QR output), `window_manager` + `tray_manager` (desktop), `file_picker`, `ffigen` (bindings to libcore).

---

## 6. Repository layout (monorepo)

```
omnitun/
├─ app/                 Flutter client (lib/, android/, windows/, ios/, macos/, linux/)
├─ core/                Go engine (libcore): protocols, tun2socks, failover, parser
│  ├─ protocol/         wireguard, awg, xray, hysteria2, tuic, stealth, ...
│  ├─ config/           importers/exporters, Profile model
│  ├─ subscription/
│  └─ bind/             C ABI + gomobile exports
├─ tor/                 Rust crate wrapping Arti, C ABI (cbindgen)
├─ platform/
│  ├─ android/          Kotlin VpnService, IKEv2, SSTP modules
│  └─ windows-svc/      Go Windows service (control protocol, Wintun, WFP, RAS)
├─ server/              Go server CLI + TUI
├─ addons/              build scripts for GPL/AGPL external binaries (openvpn, lightway, naive)
├─ proto/               shared API schema (JSON schema / protobuf)
├─ docs/
├─ scripts/             build, sign, package (MSIX/MSI, APK/AAB)
└─ .github/workflows/   CI: lint, test, build matrix
```

---

## 7. Roadmap

| Phase | Goal | Deliverables |
|---|---|---|
| **0. Bootstrap** ✅ | Toolchain and skeleton | Monorepo (single Go module at root), Rust `tor` crate, Flutter app shell with mock engine, CI, LICENSE (MIT) |
| **1. Core MVP** 🚧 | First tunnels | ✅ importer (share links, subscriptions, WG/AWG `.conf`); ✅ profile → sing-box config; ✅ in-process engine + dev CLI; ✅ desktop bridge (daemon); ✅ latency tests, live switching, traffic stats, system proxy; ✅ Android bridge (gomobile `core/mobile` + Kotlin VpnService, minSdk 29) — tested on Samsung A30s (Android 11): consent dialog, VPN traffic via tunnel, notification Disconnect, swipe-away stop, process-kill stop; ⬜ AmneziaWG engine |
| **2. Android MVP** | Real app on a phone | VpnService + permission flow, foreground notification, lifecycle stop rules, Home/Profiles/Import screens, QR scan |
| **3. Windows MVP** | Real app on desktop | ✅ `omnitun-svc` daemon + control protocol + lease + parent-death exit; ✅ Flutter DaemonEngine; ⬜ run as SYSTEM service for TUN, tray, MSIX/MSI installer |
| **4. Groups and failover** | Reliability | Groups, subscriptions with auth, health checks, no-internet detection, failover strategies, typed errors |
| **5. More protocols** | Coverage | Stealth-WG, NordLynx, Windows RAS (IKEv2/L2TP/SSTP), Android IKEv2 + SSTP |
| **6. Tor** | Censorship resistance | Rust Arti lib, obfs4/snowflake/webtunnel PTs, bridge import |
| **7. Add-ons** | GPL-family protocols | OpenVPN (UDP/TCP, XOR, TLS-wrapped), NaiveProxy, Lightway as separately downloaded add-ons |
| **8. Server CLI** | Self-hosting | `init/install/user/export/sub serve`, Bubble Tea TUI, WG/AWG/sing-box in-process, OpenVPN/IKEv2/SSTP via packages |
| **9. Polish** | Release | i18n (en, fa, ru, zh, ar), split tunneling, kill switch, routing rules, signed releases, docs |
| Later | Other OSes | iOS/macOS (NetworkExtension), Linux client |

The MVP is Phases 0–4: a Windows + Android app with WireGuard/AWG/VLESS/Trojan/SS, import, groups, subscriptions and failover.

---

## 8. Risks and open questions

- **Lightway**: the Rust implementation is AGPL-3.0 and currently supports Linux/macOS only. Windows/Android clients need porting → lowest priority.
- **Arti on Android**: size and battery cost; PT executables must ship as `lib*.so` in `jniLibs` to be executable.
- **App stores**: Google Play's VpnService policy requires declaring VPN as core functionality; some protocols may be restricted in some regions' stores → also ship via GitHub Releases / F-Droid.
- **Windows code signing**: the driver (Wintun) is already signed; our service and installer need an EV/OV certificate for SmartScreen.
- **Binary size**: Go core with sing-box (all protocol tags) ≈ 30–50 MB; use build tags so add-ons stay optional.

---

## 9. References

- Lightway (Rust, AGPL-3.0): https://github.com/expressvpn/lightway
- Arti / pluggable transports: https://blog-7551d3.pages.torproject.net/arti_110_released/ · https://docs.rs/tor-ptmgr
- Arti on Android: https://tordev.guardianproject.info/tor-on-android/arti-mobile-on-android
- hev-socks5-tunnel (MIT): https://github.com/heiher/hev-socks5-tunnel
- amneziawg-go (MIT): https://www.freshports.org/net/amneziawg-go
- TUIC crate (GPL-3.0, protocol license-free): https://rust-digger.code-maven.com/crates/tuic
- Hiddify (Flutter + sing-box, GPL-3.0): https://aur.archlinux.org/packages/hiddify
- Flutter cross-platform VPN case study: https://www.billingserv.com/blog/why-cross-platform-vpn-support-led-us-to-rebuild-our-apps-in-flutter/
- VPN UI patterns: https://nordvpn.com/blog/mobile-app-redesign-2026/ · https://nordvpn.com/zh/blog/desktop-app-redesign-2026/ · https://vp0.com/blogs/vpn-connection-switch-ui-clone-ios · https://0xacab.org/leap/bitmask-vpn/-/issues/513
