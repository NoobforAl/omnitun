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

Decision (2026-10-15): dependencies are chosen on technical merit; license compatibility is not a constraint. The **main proxy engine is sing-box** (embedded as a Go library, pinned `v1.14.3`). It already provides VLESS/Reality, VMess, Trojan, Shadowsocks, Hysteria2, TUIC, WireGuard, the TUN inbound (gVisor/system stack), DNS, routing and urltest/selector groups, which removes most of the protocol work.

| Component | Role |
|---|---|
| **sing-box** (Go lib) | VLESS/Reality, VMess, Trojan, SS, Hysteria2, TUIC, WireGuard/NordLynx, TUN, DNS, routing, groups |
| amneziawg-go | AmneziaWG (not supported by sing-box) |
| Arti (Rust) + lyrebird/snowflake/webtunnel | Tor with bridges |
| NaiveProxy | Separate binary (Chromium network stack) |
| OpenVPN | sing-box 1.14's `openvpn-client` / `openvpn-server` endpoints (`with_openvpn` tag) — in use for client and server; the OpenVPN 2.6 binary is only needed for the XOR patch |
| OpenConnect / NaiveProxy | Also built into sing-box 1.14 (`with_openconnect`, `with_naive_outbound` — naive needs cronet, so it's off for now) |
| Lightway (Rust, Linux/macOS only today) | Separate binary, low priority |
| Windows RAS / Android `Ikev2VpnProfile` | IKEv2, L2TP, SSTP (OS-native) |
| strongSwan / xl2tpd / accel-ppp (server) | Installed as system packages by the server CLI |

Omnitun is **not tied to one core** (it doesn't use Xray at all): each protocol maps to an engine adapter (sing-box, amneziawg-go, Arti, OS-native VPN, external binary). Adding a protocol beyond the list above means adding an importer + an adapter; the UI, profiles, failover and control protocol stay the same.

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
6. **Connection progress 0 → 100 %** — the core reports real stages, not a fake timer, and the UI shows a progress ring around the connect button plus the current step's label, so the user sees exactly where a connection is (or where it got stuck):

   | Stage | % | Shown as |
   |---|---|---|
   | `permission` (Android consent / Windows elevation) | 5 | "Waiting for VPN permission" |
   | `network_check` (is the device online?) | 15 | "Checking internet connection" |
   | `prepare` (build config for the selected profiles) | 25 | "Preparing configuration" |
   | `engine_start` (sing-box / AWG / Tor start) | 45 | "Starting engine" |
   | `tunnel_up` (TUN opened / proxy port listening) | 65 | "Opening tunnel" |
   | `handshake` (first URL test through the profile) | 85 | "Contacting server" |
   | `connected` (probe passed) | 100 | "Connected" |

   On failover the ring restarts at `handshake` with "Trying <next profile>". Each stage has a timeout; on failure the ring stops at that stage, turns red, and the error names the stage (e.g. "Contacting server failed: TLS error"). Stages are sent as `state.progress {stage, percent, profile}` events on the control protocol; Tor adds its bootstrap percentage inside `engine_start`.

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

One binary (`cmd/omnitun-server`) that runs sing-box as the server and manages users. Implemented:

```
omnitun-server init --host IP_OR_DOMAIN [--name N] [--reality-site HOST:PORT] [--sub-port P]
omnitun-server proto list | enable <p> | disable <p> | port <p> <port>
omnitun-server user add <name> [--days N] [--quota GB] [--note T] | list | del | enable | disable
omnitun-server user expire <name> <days|never> | quota <name> <GB> | reset <name>
omnitun-server link <name> [--qr] [--sub]      share links + terminal QR codes
omnitun-server run                             server + subscription endpoint, live reload
omnitun-server tui                             interactive manager
omnitun-server service                         systemd unit
```

- **Protocols (in-process, sing-box):** VLESS+Reality (no certificate needed; impersonates a real site), Shadowsocks-2022 (multi-user), Trojan, Hysteria2, TUIC, AnyTLS, ShadowTLS v3 (+ Shadowsocks-2022, borrowing the Reality site's handshake), NaiveProxy (the subscription pins the self-signed certificate; certificates are 90-day and renewed on start because Chromium rejects long-lived ones), OpenVPN over UDP and TCP (own CA, tls-crypt, users sign in with name + password; `.ovpn` download per user; Omnitun clients get a sing-box JSON subscription that includes OpenVPN). Trojan/Hysteria2/TUIC use a generated self-signed certificate (links carry `insecure=1`).
- **Users:** every user gets credentials for every protocol and **one subscription URL** (`http://host:port/<random path>/<token>`) that returns all links in failover order with `subscription-userinfo` (quota, expiry), `profile-update-interval` and `profile-title` headers. The Omnitun client imports it as a group.
- **Live changes:** CLI and TUI edit the same state file; `run` reloads within 2 s and removes expired users each minute.
- **TUI** (Bubble Tea): user table (status, expiry, quota, note), add / enable-disable / expiry / quota / note / new credentials / delete, per-user view with QR codes for the subscription and each link, protocol on/off and ports.
- **Storage:** one JSON file (`state.json`, mode 0600) with atomic writes — enough for personal/small servers; SQLite only if user counts grow.
- **Verified:** every generated link connects through the Omnitun client engine to the generated server (all 5 protocols, offline test); real `run` served a subscription and browsing worked through it.

Still to do on the server: see the checklist in §7 (traffic accounting, ACME certificates, HTTPS subscriptions, WireGuard/AmneziaWG/OpenVPN/IKEv2/SSTP/Tor bridges, firewall setup, installer).

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

- **Phone (planned redesign, see Phase 7)**: no bottom bar. One very simple home screen in the style of Mullvad VPN — status, the country/location of the selected server, one connect button — and other pages (servers, logs, settings) are opened from the home screen and closed with back navigation, not through a footer.
- **Phone (today)**: bottom `NavigationBar` — Home · Profiles · Logs · Settings.
- **Desktop/tablet (≥ 840 px)**: `NavigationRail` on the left with a two-pane layout (list + details).
- **Windows**: system tray icon (connect/disconnect, profile quick-switch, quit); close-to-tray is configurable.

### 5.3 Screens

| Screen | Content |
|---|---|
| Home | Big connect button with a 0–100 % progress ring and step label while connecting (see 3.3 #6), current profile chip (tap to change), status text + timer, ↑/↓ speed, IP/location after connect, failover toasts |
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
├─ app/                    Flutter client (Android + Windows; iOS/macOS/Linux scaffolded)
│  └─ android/…/kotlin/    VpnService, platform glue for the Go core
├─ core/                   Go client core
│  ├─ profile/ importer/   profile model; share links, subscriptions, .conf
│  ├─ singbox/             profiles → sing-box config
│  ├─ engine/              tunnel, handshake + progress, failover, latency
│  ├─ control/             UI ↔ core protocol (JSON lines over loopback)
│  ├─ netcheck/ subscription/
│  └─ mobile/              gomobile entry point (Android)
├─ cmd/omnitun-svc/        desktop core daemon
├─ cmd/omnitun/            developer CLI
├─ cmd/omnitun-server/     server CLI
├─ server/                 store, serverconf (inbounds/links/subscriptions), runtime, tui, qrterm
├─ tor/                    Rust crate for Arti (C ABI stub so far)
├─ tools/devserver/        local Shadowsocks server for device tests
├─ scripts/                go.ps1 (build tags), build-android-core.sh
└─ .github/workflows/      CI
```

---

## 7. Roadmap — what's done and what's left

`[x]` done (and tested) · `[ ]` left. Device tests ran on a Samsung A30s (Android 11).

**Priority (2026-10-15): mobile first.** Android, core, server and shared UI items come first; Phase 3 (Windows VPN) is deferred until everything else is done.

### Phase 0 — Bootstrap ✅
- [x] Monorepo: Go module, Rust `tor` crate, Flutter app, CI workflow, MIT license
- [x] Build scripts with sing-box feature tags (`scripts/go.ps1`, `Makefile`, `build-android-core.sh`)
- [ ] CI builds for Android APK and Windows app (CI only runs tests today)
- [ ] `THIRD_PARTY_NOTICES` generated in CI

### Phase 1 — Core
- [x] Importer: `vless` (incl. Reality), `vmess`, `trojan`, `ss` (SIP002/legacy/2022), `hy2`, `tuic`, `wireguard` links; base64 lists; WireGuard/AmneziaWG `.conf`
- [x] Profiles → sing-box config (proxy mode, TUN mode, selector + urltest groups, remote DNS as https/tls/tcp/udp)
- [x] Engine: start/stop, live switching, traffic stats, real per-profile latency tests, warn/error log forwarding
- [x] Desktop bridge: `omnitun-svc` daemon + control protocol, lease, parent-death exit
- [x] Android bridge: gomobile `core/mobile` + Kotlin `VpnService` (minSdk 29)
- [x] Importer: Clash / Clash.Meta YAML (`proxies:`), sing-box JSON (outbounds + WireGuard endpoints) — also as subscription bodies
- [ ] Importer: Xray JSON, `.ovpn`, SIP008, `naive+https://`, Tor bridge lines
- [x] AmneziaWG engine on Android (amneziawg-go 1.x params Jc/Jmin/Jmax/S1/S2/H1-H4; handshake check, traffic, failover to the next profile; dev server `tools/awgdev`) — verified on device: handshake, ping through tunnel, notification speed, failover to the next profile
- [ ] AmneziaWG 2.0 params (S3/S4, I1-I5) — reported as unsupported for now; AmneziaWG on desktop (Windows phase)
- [ ] Profile editor (create/edit a profile by hand) and export as link/QR

### Phase 2 — Android
- [x] VPN consent dialog, foreground notification with Disconnect, `onRevoke`
- [x] Stop when swiped away (`onTaskRemoved`, honours the setting); VPN survives backgrounding; process kill tears it down
- [x] Traffic exits through the tunnel (verified with a browser); DNS through the tunnel
- [x] Import from QR camera / file / clipboard (QR and file need a manual on-device check)
- [x] Release build: R8/minify with keep rules for the Go bindings, release signing (key.properties), per-ABI APKs with compressed native libs (~30 MB each, was 71 MB); verified on device
- [ ] AAB for Google Play (needs a Play signing decision)
- [x] Per-app split tunneling ("only selected" / "bypass selected", app picker with icons) — verified: bypassed Chrome showed the phone's own IP while the VPN was up
- [ ] Always-on VPN / start on boot
- [x] Notification shows profile name and live speed (verified on device)

### Phase 3 — Windows (deferred — after all other phases)
- [x] Desktop app runs on the real core in proxy mode (system proxy on/off)
- [ ] `omnitun-svc` as a Windows service (SYSTEM) for full VPN (TUN) mode without running the app as admin
- [ ] System tray (connect/disconnect, switch profile, quit), close-to-tray
- [ ] Kill switch (WFP), crash cleanup of routes/adapters
- [ ] MSIX/MSI installer, code signing
- [ ] Check the Kaspersky-style "interface-bound socket" issue in TUN mode

### Phase 4 — Reliability & UX
- [x] Saved profiles, selection, order and settings
- [x] Failover: health checks, list order = priority, same group first, no-internet detection, toast explaining the switch
- [x] Real handshake before "Connected"; dead profiles skipped while connecting
- [x] Connection progress ring 0–100 % with step labels; failure stops at the failed step
- [x] Groups (manual + one per subscription), reorder within a group
- [x] Subscriptions: auth (none/Basic/Bearer/header/URL token), usage + expiry, auto-refresh, errors shown per subscription
- [x] Plain-language errors on the Home screen (timeouts, refused connections, bad bridges, blocked Tor …); the log keeps the raw text
- [ ] Typed, translatable error messages (`NO_NETWORK`, `AUTH_FAILED`, `TLS_ERROR`, `HANDSHAKE_TIMEOUT`, …)
- [ ] More strategies: lowest latency, round-robin; per-group strategy setting
- [ ] Fetch subscriptions through the tunnel when the provider is blocked
- [ ] Secrets (subscription tokens) in the OS keystore instead of the app's JSON file
- [ ] Onboarding screens; profile sharing (QR/link) from the app

### Phase 5 — More client protocols
- [x] OpenVPN via sing-box's `openvpn-client` endpoint: `.ovpn` import (inline certs, tls-auth/tls-crypt/tls-crypt-v2, multiple remotes, comp-lzo/compress, verify-x509-name), username/password prompt in the app, sing-box JSON import; end-to-end tests (TCP cert + tls-crypt, UDP user/pass + tls-auth); verified on device through omnitun-server
- [x] Hysteria v1, AnyTLS, SSH (password/key, host-key pinning), ShadowTLS v1-3 over Shadowsocks: links (`hysteria://`, `anytls://`, `ssh://`, `ss://…?plugin=shadow-tls`), Clash and sing-box JSON import; end-to-end tests; AnyTLS and ShadowTLS verified on device through omnitun-server
- [ ] OpenVPN XOR/Stunnel obfuscation
- [x] NordLynx: `nordvpn://<access token>?country=DE&limit=10` subscription → WireGuard profiles (key from `/v1/users/services/credentials`, servers from `/v1/servers/recommendations`); tested against a mock API (no NordVPN account to test live)
- [ ] Stealth (WireGuard over TLS/WS): sing-box WireGuard `detour` through a TLS/WS outbound with UDP-over-TCP (needs a sing-box server), later a wstunnel-compatible Go client
- [x] SSTP and IKEv2 in the core on pure-Go `veepin` (patched copy in `third_party/veepin`, see PATCHES.md: app-supplied TUN; EAP with a certificate-authenticated server). `sstp://` and `ikev2://` links (IKEv2: PSK, EAP-MSCHAPv2, certificates, IKE over TCP/RFC 8229). The app's own sockets stay off the tunnel (`exclude_self`). Tests: in-process veepin servers (SSTP, IKEv2 PSK + EAP over TCP) with a ping through the tunnel; live SSTP handshake + ping against VPN Gate; on device: SSTP to a dev server (`tools/sstpdev`) with ping through the tunnel; VPN Gate SSTP connects but that public server routes replies to a stale session
- [x] IKEv2 on the OS client (`system=1`, Android 11+ `Ikev2VpnProfile`) as an option — note many phones (e.g. Samsung A30s) lack `FEATURE_IPSEC_TUNNELS`, which is why the in-core engine is the default
- [ ] IKEv2 with a certificate-authenticated server + EAP: implemented in the fork but not yet tested against a real strongSwan server
- [ ] L2TP/IPsec: deferred (possible in userspace — TunnelForge does it — but low value)
- [ ] Windows built-in VPN: IKEv2, L2TP/IPsec, SSTP (Windows phase)
- [x] NaiveProxy via sing-box `with_naive_outbound` (Chromium cronet, statically linked on Android: +5 MB per APK): `naive+https://` / `naive+quic://` links, sing-box JSON with a pinned certificate; e2e test with real cronet on Windows (libcronet.dll); verified on device through omnitun-server
- [ ] Lightway (Rust `lightway-core`, AGPL; only useful with self-hosted servers — low priority)

### Phase 6 — Tor
- [x] Rust crate with C ABI (stub)
- [x] `omnitun-tor` binary (Rust, `tor/`): Arti 0.47 (rustls + ring, static sqlite) behind a minimal local SOCKS5 port; JSON progress lines on stdout (`listening`, `bootstrap` %, `ready`, `error`); exits when stdin closes; bridge lines and unmanaged transports as arguments; unit tests (args, SOCKS)
- [x] `core/tor`: bridge pluggable transports in Go (lyrebird 0.8.1: obfs4, WebTunnel, Snowflake, meek), each on a local PT-SOCKS port for Arti; process manager (start, progress, ready, stop); Tor Browser's built-in bridges embedded (obfs4, Snowflake, meek); bridge-line parsing; opt-in live test (`OMNITUN_LIVE_TOR=obfs4|snowflake|meek|direct`, `OMNITUN_TOR_BIN`)
- [x] Engine: `tor` profile (`tor://`, `tor://snowflake`, `tor://obfs4`, pasted bridge lines): Tor starts first and its bootstrap fills the progress ring ("Connecting to Tor NN%", then "Building a Tor circuit"); the tunnel only opens after a real test connection through a fresh circuit works; up to 3 starts with fresh guard state (Snowflake is intermittent); sing-box routes the profile into Arti's SOCKS port; failover to the next profile if Tor can't connect; Tor profiles only run when selected
- [x] Fail closed: a Tor profile never fails over to a non-Tor profile on its own, and the Tor client never logs the destinations it connects to
- [x] Android packaging: `scripts/build-android-tor.sh` builds `omnitun-tor` with `cargo ndk` into `jniLibs/<abi>/libomnitun_tor.so` (6.2 MB per ABI); the core gets its path through `mobile.SetTorBinary`; the app is kept out of its own VPN so Tor and the bridge transports reach the network directly
- [x] **Verified on the A30s in Iran (Tor blocked directly):** Tor over built-in Snowflake in proxy mode and in full VPN mode; Firefox through the tunnel reached `check.torproject.org` and reported `IsTor: true`. Findings that shaped the code: Arti keeps guards marked "down" between runs, so guard state is reset on every connect (with a separate directory per profile); the app being excluded from the VPN breaks sing-box's system TCP stack, so Tor mode uses the gVisor stack; Tor exits don't allow DNS over TLS (port 853), so DNS goes over DoH while Tor is active; Tor has no UDP, so QUIC falls back to TCP
- [ ] Built-in obfs4 bridges were all unreachable from that network (blocked), and Arti rejects bridge lines without a fingerprint, which includes Tor Browser's built-in meek line (meek is not offered for now); custom bridges from bridges.torproject.org should work, untested
- [ ] App UI: bridge settings screen (none / Snowflake / obfs4 / custom lines, QR and file import) — today the Import sheet has a "Tor" quick-add menu only; Tor exit country/IP shown after connecting; quiet Snowflake's verbose logging
- [ ] Desktop (Windows): run `omnitun-tor` next to the service; Tor on other ABIs (armeabi-v7a, x86_64) is built by the release script but untested

### Phase 7 — UI redesign (Mullvad-style, very simple)
- [ ] Home like Mullvad VPN: dark, calm, almost empty; shows the connection state (honest, as today) and the **country** (flag + country/city name) of the server in use, plus one big connect/disconnect button
- [ ] **No footer / bottom navigation bar**: the other pages (server list, logs, settings) open from icons or the location row on the home screen and close with back; nothing to switch between at the bottom
- [ ] Location picker like Mullvad: countries → cities → servers (profiles), with the flag and a latency dot; "Auto" on top; the same screen holds groups/subscriptions
- [ ] Country for every profile: from the server's GeoIP (offline database, or a lookup through the tunnel after connecting) and from names like "DE-Berlin-01"; manual override in the profile editor; Tor shows the exit country once connected
- [ ] Advanced things (protocol, logs, split tunneling, DNS, bridges) stay out of the main flow, inside Settings
- [ ] Same design on desktop (one narrow window / tray popup) and Android; keep the failover toast and the connection progress ring (Phase 4)

### Phase 8 — Server (`omnitun-server`): CLI, TUI and web API
**Order (2026-10-16): the server comes first, then the protocol layer, then the app GUI.** The GUI (Phase 7) waits until the server and the shared protocol interface are done.

Done so far
- [x] `init` with Reality keys, Shadowsocks-2022 key, self-signed certificate
- [x] Users: add / list / delete / enable / disable / expiry / quota / note / new credentials
- [x] Protocols: VLESS+Reality, Shadowsocks-2022, Trojan, Hysteria2, TUIC, AnyTLS, ShadowTLS v3, NaiveProxy, OpenVPN UDP/TCP (all verified end-to-end with the Omnitun client)
- [x] Per-user links + terminal QR codes; one subscription URL per user with quota/expiry headers
- [x] `run` with live reload on changes and expiry; `service` prints a systemd unit
- [x] TUI: users, links + QR, protocol on/off and ports

Step 1 — one service layer for every front end
- [x] `server/manager`: every operation (init, protocols, users, links, status, reload) as typed methods with JSON-tagged results; the CLI, the TUI and the web API only call it, so they always behave the same
- [x] **CLI on cobra** (help, shell completion, `--json` output, `--data`, consistent errors); same commands as today
- [x] **Web API on huma + chi** (OpenAPI 3 generated, validation, `/docs`): `/api/v1/server`, `/protocols`, `/users`, `/users/{name}` (get, patch, delete), `/users/{name}/reset`, `/users/{name}/links`, `/users/{name}/ovpn/{proto}`, `/reload`; bearer token (`omnitun-server api token`), listens on 127.0.0.1 by default, optional TLS; started by `run`
- [x] **TUI on Bubble Tea** (+ bubbles, huh form for adding users): same features through the manager, tabs for Users / Protocols / Server, API token view; per-user traffic and logs come with Step 2
- [x] Tests for the manager, the API (httptest, incl. live token rotation) and the TUI; CLI checked by hand (exit codes 2/3/4 for invalid/not found/conflict)
- [x] CLI tests (cobra command tree against a temp data dir)

Step 2 — accounting and safety
- [x] Per-user traffic accounting and quota **enforcement**: sing-box connection counters summed per user (`server/stats`, `stats.json`), monthly quota (UTC calendar month, GB = 2^30 bytes) blocks the user until the month rolls over, the quota is raised or `user usage <name> --reset`; shown in CLI (`user list`, `user usage`), TUI, API (`usage` on users, `POST /users/{name}/usage/reset`) and in the subscription `Subscription-Userinfo` header; covered for VLESS, Trojan, Shadowsocks/ShadowTLS, Hysteria2, TUIC, AnyTLS, Naive (**not OpenVPN yet**: its endpoint does not report the user)
- [x] File locking (`state.json.lock`, works across processes): CLI, TUI, API and the running server can write at the same time without losing changes
- [x] Backup and restore: `omnitun-server backup <file>` / `restore <file> [--force]` (tar.gz of state, keys and certificates; hostile archives rejected) and `GET /api/v1/backup`
- [x] Live status: running/not running, connections and speed per user (CLI `status`/`user list`, TUI, API `server`)
- [ ] Traffic per protocol and server logs (API + TUI)
- [ ] OpenVPN per-user accounting (read the endpoint's status/management output)

Step 3 — more protocols and setup
- [ ] **Shared protocol registry** used by both the server and the client: one definition per protocol (name, transports, default port, server options, client profile fields, link format); "create a server" and "connect to a server" use the same schema, so adding a protocol is one place
- [ ] ACME/Let's Encrypt certificates for a domain (no more `insecure=1`); HTTPS for the subscription endpoint
- [ ] WireGuard / AmneziaWG server, Tor bridge (obfs4/WebTunnel), IKEv2 / SSTP via system packages
- [ ] Easy v2ray-style setup: `init` wizard (choose protocols, domain, ports), import an existing v2ray/xray config and its users, one-command install
- [ ] Firewall/NAT setup (nftables), `install` command that writes and enables the systemd unit

### Phase 9 — Polish & release
- [ ] Translations (en, fa, ru, zh, ar) with right-to-left layout
- [ ] Routing rules (geoip/geosite, bypass LAN, per-domain), IPv6 toggle, MTU
- [x] Local networks bypass the VPN by default (Settings → "Bypass local network"): 10/8, 172.16/12, 192.168/16, link-local, multicast, broadcast and IPv6 ULA/link-local are left out of the VPN routes for every engine (sing-box `route_exclude_address`, AmneziaWG/SSTP/IKEv2 route ranges) and sent direct; a tunnel's own subnet, DNS and explicitly routed WireGuard subnets stay inside; verified on device (`ip route` on tun0)
- [ ] Bypass list from a .txt file: the user imports (file picker / paste / URL) a list of domains, IPs and IP ranges that go direct instead of through the VPN — one entry per line: `example.com` (and its subdomains), `full:exact.example.com`, `keyword:bank`, `regexp:…`, `1.2.3.4`, `10.0.0.0/8`, `2001:db8::/32`; `#` comments; several named lists, each on/off; optional "proxy only these" mode (inverse list); mapped to sing-box route rules (`domain_suffix`, `domain`, `domain_keyword`, `domain_regex`, `ip_cidr` → direct), and on Android IP ranges are also excluded from the VPN routes
- [ ] Signed releases on GitHub (APK, Windows installer, server binaries for Linux amd64/arm64), F-Droid
- [ ] User docs

### Later
- [ ] iOS / macOS (NetworkExtension), Linux desktop client

---

## 8. Risks and open questions

- **Lightway**: the Rust implementation is AGPL-3.0 and currently supports Linux/macOS only. Windows/Android clients need porting → lowest priority.
- **Arti on Android**: size and battery cost; PT executables must ship as `lib*.so` in `jniLibs` to be executable.
- **App stores**: Google Play's VpnService policy requires declaring VPN as core functionality; some protocols may be restricted in some regions' stores → also ship via GitHub Releases / F-Droid.
- **Windows code signing**: the driver (Wintun) is already signed; our service and installer need an EV/OV certificate for SmartScreen.
- **Binary size**: Go core with sing-box (all protocol tags) ≈ 30–50 MB; use build tags so add-ons stay optional.

---

**Dependency note:** do not run `go get -u ./...` blindly. Snowflake (via lyrebird) needs `pion/webrtc` v4.2.18 with `pion/transport` v4, and the latest pion releases move to transport v5, which breaks the build. Upgrade selected modules (e.g. `go get github.com/sagernet/sing-box@vX`) and run the tagged tests (`scripts/go.ps1 test ./...`).

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
