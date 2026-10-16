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
| NaiveProxy | sing-box `with_naive_outbound` (Chromium cronet, statically linked on Android; libcronet.dll on Windows) |
| OpenVPN | sing-box 1.14's `openvpn-client` / `openvpn-server` endpoints (`with_openvpn` tag) — in use for client and server; the OpenVPN 2.6 binary is only needed for the XOR patch |
| OpenConnect | Built into sing-box 1.14 (`with_openconnect`); not exposed in the app yet |
| SSTP / IKEv2 | Pure-Go `veepin` in the core (patched copy in `third_party/veepin`); OS IKEv2 client optional |
| Lightway (Rust, Linux/macOS only today) | Separate binary, low priority |
| Windows RAS / Android `Ikev2VpnProfile` | IKEv2, L2TP, SSTP (OS-native) |
| strongSwan / xl2tpd / accel-ppp (server) | Installed as system packages by the server CLI |
| sing-box `tailscale` endpoint (`with_tailscale`) | Planned (Phase 10): peer-to-peer mesh with a Tailscale or Headscale control server |
| Nebula (in `third_party/veepin`) / WireGuard + our own coordinator | Planned (Phase 10): a mesh that needs no third-party control server |
| DNSCrypt library (`github.com/ameshkov/dnscrypt/v2`, `dnsstamps`) | Planned (Phase 11): DNSCrypt v2 and Anonymized DNSCrypt; sing-box 1.14 has no DNSCrypt transport |

Omnitun is **not tied to one core** (it doesn't use Xray at all): each protocol maps to an engine adapter (sing-box, amneziawg-go, Arti, OS-native VPN, external binary). Adding a protocol beyond the list above means adding an importer + an adapter; the UI, profiles, failover and control protocol stay the same.

IKEv2, L2TP/IPsec and SSTP on **Windows** use the built-in RAS VPN client (no third-party code). On **Android**: IKEv2 uses the platform `Ikev2VpnProfile` (Android 11+); SSTP and IKEv2 also run in the core (pure-Go `veepin`), so they work on phones without IPsec support; **L2TP is not available** (Android 12 removed it) — the UI shows "not supported on this device".

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

### 2.4 Peer-to-peer mesh VPN (plan, Phase 10)

What "p2p like a VPN" means here: devices reach **each other** directly (phone ↔ home PC ↔ a friend's phone) over encrypted WireGuard tunnels, with no VPN server in the data path when the NATs allow it; and any device can offer to be an **exit** for the others ("use my home connection"). Like Tailscale, ZeroTier or Nebula, not like a file-sharing P2P network.

Three layers, built in this order so each is useful alone:

1. **Join an existing mesh (client only).** A `tailscale://` profile (auth key, optional Headscale/control URL, optional exit node) runs sing-box's `tailscale` endpoint (it embeds the Tailscale client: control, DERP relays, NAT traversal, MagicDNS). Cheapest path; works with Tailscale or a self-hosted **Headscale**. Needs no new server code.
2. **Our own coordinator in `omnitun-server`** (so a mesh does not depend on anyone's control plane, which matters where Tailscale is blocked): the server keeps the device list per user, hands each device a WireGuard key, a mesh address (a private range of our own, not the 10/8, 172.16/12, 192.168/16 ranges the VPN NAT drops) and the list of peers with their latest endpoints; devices find each other's public address with **STUN**, punch through NATs by sending to each other at the same time, and fall back to a **relay** (the server forwards the already-encrypted packets) when both sides are behind symmetric NAT or CGNAT. Data plane: WireGuard (sing-box's endpoint or amneziawg-go, so the mesh can also be obfuscated); control plane: HTTPS long-poll over the same TLS certificate and subscription port.
3. **Exit through a friend's device.** An invite link (`omnitun://mesh?…`) gives one person the right to use my home PC or phone as an exit: opt-in on the sharing side, per-peer bandwidth and monthly caps, allow/deny lists, a visible "sharing" notification on Android, and revoke at any time. The exit device NATs the guest's traffic and drops private ranges and other guests (same rules as the server's NAT).

Open questions before layer 2: ACLs (who may reach whom; start with "all my devices see each other, guests see only the exit"), DNS names for devices (`phone.mesh`), IPv6, and whether a phone may be an exit at all (battery, mobile data, store policy; default off).

Another reading of "p2p" is a **volunteer network** (Snowflake/Lantern style: strangers donate bandwidth to people in censored countries). It is a different product (abuse handling, legal exposure, broker infrastructure) and is not planned here; the Snowflake client we already ship is the supported way to use one.

### 2.5 DNSCrypt (plan, Phase 11)

[DNSCrypt](https://www.dnscrypt.org/) encrypts **and authenticates** DNS between the client and the resolver (the resolver's public key is part of its address, so a man in the middle cannot answer), over UDP or TCP, and **Anonymized DNSCrypt** hides the client's address from the resolver by sending the query through a relay. Today the app offers DoH/DoT/TCP/UDP through one "Remote DNS" URL (`core/singbox/build.go` `dnsServer`).

Design:
- **Addresses are DNS stamps** (`sdns://…`, a compact string that holds the protocol, address, port, provider name and public key; it also covers DoH, DoT and DoQ resolvers and relays). The core parses them (`github.com/ameshkov/dnsstamps`), and the existing "Remote DNS" field accepts a stamp as well as a URL.
- **A local DNS forwarder in the core** speaks DNSCrypt and listens on loopback (UDP and TCP); sing-box's DNS server points at it (`type: udp`, `127.0.0.1:port`) because sing-box has no DNSCrypt transport. The forwarder's own sockets go **through the tunnel** (the profile's outbound as detour) so the network sees only the VPN; with Tor it uses TCP through Arti's SOCKS port (Tor has no UDP). It fetches and caches the resolver's certificate, rotates when it expires, and refuses a resolver whose key does not match the stamp.
- **Resolver list:** a snapshot of the public `public-resolvers.md` list (stamps, with the minisign signature checked) is bundled in the app, because GitHub (where the list lives) may be blocked; it is refreshed through the VPN. The picker shows a short curated set (no-logging, no-filtering, DNSSEC) with a country and latency, "Anonymized (through a relay)" as a switch, and "Custom stamp".
- **Server side (optional):** `omnitun-server` can run a DNSCrypt resolver for its own users (same library), so the VPN's DNS is private and under the operator's control; keys are generated at `init` and rotated; the port is opened with `firewall`. Off by default.
- **Failure behaviour:** if the DNSCrypt resolver cannot be reached the connection does not silently fall back to plain DNS; the home screen says "DNS unavailable" and offers the previous resolver (a setting, off = fail closed).

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

Split tunneling (per-app on Android, per-route on Windows), bypass LAN, geoip/geosite rules, custom DNS (DoH/DoT now, DNSCrypt in Phase 11), IPv6 on/off, MTU, latency test for all profiles, traffic statistics, logs view with export.

---

## 4. Server (`omnitun-server`, Go, Linux first)

One binary (`cmd/omnitun-server`): CLI (cobra), TUI (Bubble Tea) and a web API (huma + chi) over one service layer (`server/manager`); the VPN core is sing-box in-process.

```
omnitun-server init --host IP_OR_DOMAIN [--name N] [--reality-site HOST:PORT] [--sub-port P]
omnitun-server status                          running state, users, connections
omnitun-server proto list | enable <p> | disable <p> | port <p> <port>
omnitun-server user add <name> [--days N] [--quota GB] [--note T]
omnitun-server user list | del | enable | disable | expire | quota | note | reset | usage [--reset]
omnitun-server link <name> [--qr] [--sub] [--ovpn udp|tcp]
omnitun-server api status | enable | disable | token [--new] | listen <host> [port]
omnitun-server backup <file> | restore <file> [--force]
omnitun-server run                             VPN core + subscriptions + web API (if enabled), live reload
omnitun-server tui                             interactive manager
omnitun-server service                         systemd unit
(every command: --data DIR, --json; exit codes 2 invalid / 3 not found / 4 conflict)
```

- **Protocols** come from one registry (`core/protocols`, shared with the client): VLESS+Reality, Shadowsocks-2022, Trojan, Hysteria2, TUIC, AnyTLS, ShadowTLS v3, NaiveProxy (the subscription pins the self-signed certificate; certificates are 90-day and renewed on start), OpenVPN UDP/TCP (own CA, tls-crypt, name + password, `.ovpn` per user). Trojan/Hysteria2/TUIC/AnyTLS use a generated self-signed certificate (links carry `insecure=1`).
- **Users:** every user gets credentials for every protocol and **one subscription URL** returning all links in failover order, with `Subscription-Userinfo` (real monthly usage, quota, expiry), `profile-update-interval` and `profile-title`. The Omnitun client imports it as a group.
- **Accounting:** per-user traffic (month and lifetime), live connections and speed (`stats.json`); the monthly quota blocks a user at the cap until the month rolls over, the quota is raised or the usage is reset.
- **Web API:** OpenAPI at `/openapi.json`, docs at `/docs`, bearer token read live (rotation or disable applies within ~2 s); loopback by default.
- **Storage:** `state.json` (0600), atomic writes under a cross-process file lock; backup/restore as tar.gz.
- **Verified:** every generated profile connects through the Omnitun client engine to the generated server (all protocols, offline test); traffic is counted end to end through a real client connection; Tor/others verified on the phone.

Still to do on the server: see Phase 8 (the real-VPS checks), Phase 10 (mesh coordinator and relay) and Phase 11 (optional DNSCrypt resolver). OpenVPN accounting, live numbers and the Docker test are done.

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

- **Every size (done, Phase 7)**: no bottom bar and no side rail. One very simple home screen in the style of Mullvad VPN — status ring, the country of the server in use, a location row and one Connect button — and the other pages (locations, logs, settings, manage servers) open from it and close with Back. Wide windows show the same screen as one centered column (max 480 px), which is also the layout of a narrow "mini" window.
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

- Material 3 with a hand-made **dark-fantasy palette in cold, muted blues** (night-slate surfaces, steel-blue actions; light theme = cold stone); dark by default. Fonts are open (SIL OFL), bundled and cut to the scripts used: **Cinzel** for headings, **Inter** for text, **Vazirmatn** for Persian and Arabic (`app/assets/fonts`, `scripts/gen-fonts.py`).
- Semantic colors: connected = frost-teal, connecting = worn gold, error = dried-blood red, offline = grey (never by color alone).
- Home background is a **dotted Earth in the manner of Mullvad's**: it shows where you are while unprotected and flies to the exit when you connect (see Phase 7). No text is written on it.
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
│  ├─ profile/ importer/   profile model; share links, subscriptions, configs → profiles
│  ├─ links/               profiles → share links (reverse of importer)
│  ├─ protocols/           protocol registry shared with the server (Spec, server inbound, client profile)
│  ├─ singbox/             profiles → sing-box config
│  ├─ engine/              tunnel, handshake + progress, failover, latency, Tor/AWG/SSTP/IKEv2 runs
│  ├─ awg/ classicvpn/     AmneziaWG, SSTP + IKEv2 (veepin)
│  ├─ tor/                 Arti process manager + bridge transports
│  ├─ nordlynx/ subscription/ netcheck/
│  ├─ control/             UI ↔ core protocol (JSON lines over loopback)
│  └─ mobile/              gomobile entry point (Android)
├─ cmd/omnitun-svc/        desktop core daemon
├─ cmd/omnitun/            developer CLI
├─ cmd/omnitun-server/     server CLI (cobra)
├─ server/                 manager (service layer), api (huma), tui, store, stats, serverconf, runtime, qrterm
├─ tor/                    Rust: omnitun-tor (Arti behind a local SOCKS5 port)
├─ third_party/veepin/     patched pure-Go SSTP/IKEv2 (see PATCHES.md)
├─ third_party/sing-openvpn/  patched copy: reports which user holds which tunnel address (see PATCHES.md)
├─ tools/                  devserver, awgdev, sstpdev: local servers for device tests
├─ tools/dockertest/       `make docker-e2e`: the Linux server against real clients in Docker
│                          (planned: core/mesh, server/mesh, core/dnscrypt; see Phases 10 and 11)
├─ scripts/                go.ps1 (build tags), build-android-core.sh, build-android-tor.sh, build-android-release.sh
└─ .github/workflows/      CI
```

---

## 7. Roadmap — what's done and what's left

`[x]` done (and tested) · `[ ]` left. Device tests ran on a Samsung A30s (Android 11).

**Priority (2026-10-15): mobile first.** Android, core, server and shared UI items come first; Phase 3 (Windows VPN) is deferred until everything else is done.

### Status board (2026-10-17)

**Working today (tested; on-device parts on a Samsung A30s, Android 11; the Linux-only server parts in Docker on Linux with real clients, see below):** the client core with 20+ link/config formats; the Android app with VpnService, failover, subscriptions (also through the VPN), split tunneling, bypass lists, local-network bypass, always-on VPN, release APKs; the **Mullvad-style UI** (one screen, no footer, country list, Automatic, exit address and country on request), profile editor, share/QR, onboarding, **five languages with right-to-left layout**; protocols WireGuard, AmneziaWG 1.x, OpenVPN, VLESS/VMess/Trojan/Shadowsocks/Hysteria/Hysteria2/TUIC/AnyTLS/ShadowTLS/SSH/Naive, SSTP, IKEv2, NordLynx subscriptions and Tor (Snowflake verified in Iran); the server with 14 protocols (SSTP, IKEv2 and AmneziaWG off by default), users, quotas, per-user accounting and live connections/speed for every protocol, CLI, TUI, web API, backup, ACME, wizard, Xray import and installer; one protocol registry shared by server and client; user docs, CI, open-source licenses inside the app, and `make docker-e2e`.

**Docker end-to-end test (`make docker-e2e`, `tools/dockertest/`):** the Linux server binary runs in a container with a kernel TUN device and NAT, and a fresh container per protocol connects with the real client: `openvpn` (UDP and TCP), `wg-quick`, `sstpc` + pppd (SSTP), strongSwan (IKEv2), and our AmneziaWG client on a kernel TUN. All six pass: traffic through the tunnel leaves the server NAT-ed, the server shows the user's live connection and speed while a download runs, and the download is counted for the right user. **It found three real bugs that no fake had shown** (all fixed): the SSTP/IKEv2 TUN devices were never addressed or brought up; the SSTP server wrote plain packets to a GSO device (every write refused with EINVAL); the IKEv2 server could not receive bare ESP from a client without NAT (now it always makes the client use UDP encapsulation). The other server protocols run through sing-box and were already tested on this PC.

**Not verified (cannot be done from this PC, or needs a person):**

* a **real Linux VPS** (Docker is a Linux VM, not the internet): `install`/`firewall` (needs systemd and ufw/firewalld), ACME against a real CA, arm64 binaries; SSTP against the **Windows** client and IKEv2 against **Windows/iOS/Android** clients (only `sstpc` and strongSwan were tried; Windows is picky about certificates and algorithms); a client behind a real NAT (the test network has none); the Docker test covers one user at a time and IPv4 only;
* on the phone: QR scanning with the camera, importing a file through the system picker (my automation could not select a file), onboarding (the phone has servers), "only these" bypass mode, address-range bypass, Chinese and Arabic screens, the lockdown switch ("Block connections without VPN"), Tor on armeabi-v7a/x86_64, custom Tor bridge lines;
* the translations were written by me and need a native reader (the newest: editor/import/subscription/bypass messages, Tor steps, country names, licenses screen); the new country names, translated messages, licenses screen and Snowflake log filter were **not looked at on the phone** (tests only); the GitHub workflow has never run.

**What is left, in the order I would do it**

1. **Read, try and correct** what is listed as not verified above (a person with a Linux VPS and the phone: about an hour).
2. **Release**: a GitHub release job (signed APKs, server binaries, notices; needs the signing key as a secret), CI builds for APK and Windows, F-Droid; upload to Google Play (needs a signing and listing decision).
3. **Server**: run the Docker test in CI (a job with the Docker socket), more than one user at a time, IPv6.
3b. **New (2026-10-17), planned only, nothing written yet:** **Phase 11 DNSCrypt** first (small, self-contained, helps every user), then **Phase 10 peer-to-peer mesh** (large; layer 1 needs no server code).
4. **Protocol layer**: OpenVPN XOR/Stunnel and AmneziaWG 2.0 (both blocked upstream: sing-box 1.14.3 and amneziawg-go 1.0.4), Stealth (WireGuard over TLS), Lightway (low priority), L2TP (deferred).
5. **App polish**: an app lock with the keystore decision (see Phase 4).
6. **Windows (deferred by decision)** — Phase 3: service for TUN mode, tray (the "mini window" popup), kill switch, installer and signing.
7. **Later** — iOS/macOS (NetworkExtension), Linux desktop client.

**Known limits and debt:** pion pinned for Snowflake (see the dependency note); built-in obfs4/meek bridges unusable from Iran (custom bridges untested); OpenVPN per-user accounting depends on a patched copy of sing-openvpn (`third_party/sing-openvpn`, see its `PATCHES.md`) and must be re-applied when upgrading it; "connected" for an AmneziaWG user means a handshake in the last 3 minutes (WireGuard has no sessions); IKEv2 can miss the traffic of the last 2 s before a client leaves; messages written by the core that have no code (mostly low-level network errors) are shown in English; the app has no GeoIP database (country = what the user set, what the exit lookup saw, or a guess from the name).

### Phase 0 — Bootstrap ✅
- [x] Monorepo: Go module, Rust `tor` crate, Flutter app, CI workflow, MIT license
- [x] Build scripts with sing-box feature tags (`scripts/go.ps1`, `Makefile`, `build-android-core.sh`)
- [x] CI (`.github/workflows/ci.yml`): Go vet/test (Linux and Windows), Rust, Flutter analyze/test, a `server-release` job (static Linux amd64/arm64 server binaries with SHA256SUMS, `scripts/build-server-release.sh`) and a `notices` job (below). **The workflow was written and its commands were run locally, but it has not run on GitHub**
- [ ] CI builds for the Android APK and the Windows app (the Android release needs the NDK, gomobile and the signing key as secrets; `scripts/build-android-release.sh` does it by hand)
- [x] `THIRD_PARTY_NOTICES.md` generated by `scripts/gen-notices.py` from the modules linked into the binaries, `app/pubspec.lock` and `tor/`'s crates: 693 packages, every license identified (MIT, Apache-2.0, BSD, ISC, MPL, Unicode, and GPL-3.0 for sing-box and its `sing-*` libraries), one text per distinct license (~1 MB); the CI `notices` job fails when a license cannot be identified and keeps the file as an artifact. **Not shown inside the app yet** (Flutter's own license page would only list the Dart packages)

### Phase 1 — Core
- [x] Importer: `vless` (incl. Reality), `vmess`, `trojan`, `ss` (SIP002/legacy/2022), `hy2`, `tuic`, `wireguard` links; base64 lists; WireGuard/AmneziaWG `.conf`
- [x] Profiles → sing-box config (proxy mode, TUN mode, selector + urltest groups, remote DNS as https/tls/tcp/udp)
- [x] Engine: start/stop, live switching, traffic stats, real per-profile latency tests, warn/error log forwarding
- [x] Desktop bridge: `omnitun-svc` daemon + control protocol, lease, parent-death exit
- [x] Android bridge: gomobile `core/mobile` + Kotlin `VpnService` (minSdk 29)
- [x] Importer: Clash / Clash.Meta YAML (`proxies:`), sing-box JSON (outbounds + WireGuard endpoints) — also as subscription bodies
- [x] Importer: `.ovpn`, `naive+https://`, Tor bridge lines and `tor://` links
- [x] Importer: Xray/V2Ray client JSON (vless/vmess/trojan/shadowsocks/wireguard outbounds; ws/grpc/httpupgrade/h2, TLS and Reality; both vnext and flat layouts; xhttp reported as unsupported) and SIP008; an Xray *server* config is refused with a hint to use `omnitun-server import xray`
- [x] AmneziaWG engine on Android (amneziawg-go 1.x params Jc/Jmin/Jmax/S1/S2/H1-H4; handshake check, traffic, failover to the next profile; dev server `tools/awgdev`) — verified on device: handshake, ping through tunnel, notification speed, failover to the next profile
- [x] AmneziaWG 1.5 parameters (I1-I5 signature packets, J1-J3 controlled junk): passed to amneziawg-go and accepted by the device (tested); a link or `.conf` carrying them round-trips
- [ ] AmneziaWG 2.0 (S3/S4, header ranges like `H1=100-200`) — blocked upstream: amneziawg-go 1.0.4 (the latest release) has no such keys, so they are reported as unsupported; AmneziaWG on desktop (Windows phase)
- [x] Export a profile as a share link (`core/links`, round-trip tested)
- [x] Profile editor in the core: `profile.Forms()` describes the form of 16 protocols (fields, kinds, choices, required, advanced), `FromValues`/`Apply`/`Values` build, edit and read profiles, and the control protocol serves it (`forms`, `build_profile`, `profile_values`, `export_link`). Tests: every protocol builds from typed values, exports a link, re-imports to the same values and (sing-box protocols) builds an outbound
- [x] Profile editor and share/QR screens in the app: "Add a server by hand" (location list "+" menu) picks a protocol, then a form built from the core's `forms` (required marks, switches, choices, secrets with a show toggle, "More options" for the advanced fields, a country row); Save goes through the core (`build_profile`), whose messages are shown as they are; "Edit" in a server's menu changes it in place (same group and place, what the form does not show is kept); "Share link or QR" shows the share link as a QR code and copies it (`qr_flutter`). Tested: widget tests with the mock engine and a Dart test against the real `omnitun-svc` (forms, build, edit, read back, export, re-import); on the A30s the editor and the QR screen were opened with the real core. **Not tested:** scanning that QR with a camera, and editing every protocol's form on the device

### Phase 2 — Android
- [x] VPN consent dialog, foreground notification with Disconnect, `onRevoke`
- [x] Stop when swiped away (`onTaskRemoved`, honours the setting); VPN survives backgrounding; process kill tears it down
- [x] Traffic exits through the tunnel (verified with a browser); DNS through the tunnel
- [x] Import from QR camera / file / clipboard / typed text. On the A30s with the new UI: typed text imported (and was deleted again); the system file picker opens, but in my automation this phone's picker did not accept a tap on a .txt file, so **file import and the QR camera still need a manual check**
- [x] Release build: R8/minify with keep rules for the Go bindings, release signing (key.properties), per-ABI APKs with compressed native libs (~30 MB each, was 71 MB); verified on device
- [x] AAB: `flutter build appbundle --release` builds (54 MB for arm64 here, signed with the release key). **Not uploaded to Google Play** (needs a Play signing and store-listing decision); the full multi-ABI bundle needs the core built for all ABIs (`scripts/build-android-release.sh`)
- [x] Per-app split tunneling ("only selected" / "bypass selected", app picker with icons) — verified: bypassed Chrome showed the phone's own IP while the VPN was up
- [x] Always-on VPN / start on boot: the app saves its last "start" request when it connects in VPN mode; when Android starts the VPN service by itself (Always-on VPN at boot or after the app was killed) the service starts the core and reconnects with that request (`Mobile.StartTunnelFromJSON`, `OmnitunVpnService.reconnectSaved`). **Verified on the A30s:** with the app killed, switching "Always-on VPN" on in Android's VPN settings started the service without any UI and brought Tor (Snowflake) up; a manual Disconnect afterwards stays disconnected. Settings has a tile that opens Android's VPN settings for "Always-on VPN" and "Block connections without VPN" (that system switch is the kill switch; **not tested**; connect once from the app first or there is nothing to reconnect to). The old "Kill switch" toggle did nothing and is gone
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
- [x] Typed, translatable error messages: the core's state carries `error_code` next to its English sentence (`NO_NETWORK`, `TIMEOUT`, `REFUSED`, `RESET`, `NO_ROUTE`, `NETWORK_UNREACHABLE`, `DNS_FAILED`, `TLS_ERROR`, `AUTH_FAILED`, `CLOSED`, `TOR_TIMEOUT`, `TOR_BRIDGE_TIMEOUT`, `TOR_BAD_BRIDGE`, `TOR_MISSING`; one table makes the code and the sentence agree), the app adds `NO_PERMISSION` and `CORE_LOST`, and the home screen words them as "<stage> failed: <reason>" in the user's language; anything unusual is shown as the core wrote it (English). Tested in the core, and for every language in the app
- [x] Strategies: "Automatic" in the location list tries the fastest known server first (from the speed test) and fails over in that order; round-robin and a per-group strategy setting were dropped on purpose (one tunnel carries all traffic, and failover already covers availability)
- [x] Fetch subscriptions through the tunnel when the provider is blocked: a subscription has a "Download" choice: directly, **directly or through the VPN if the provider is blocked** (default; used only when the direct fetch fails and the VPN is up, never after a wrong password), or always through the VPN. The core dials through the running sing-box tunnel's selected outbound (`Engine.TunnelDialer`, `subscription.FetchVia`, request field `via`; NordLynx subscriptions too). Tested end to end (an in-process Shadowsocks server whose route maps a made-up provider name to a local web server: unreachable directly, fetched through the tunnel), in the control protocol and from Dart against the real daemon. **Not tried on the phone**; AmneziaWG, SSTP and IKEv2 tunnels cannot carry it (the core says so)
- [ ] Secrets (server passwords/keys, subscription tokens) in the OS keystore instead of the app's JSON file: **decided not now**. The file lives in the app's private storage, and Android backup of it is now off (that was the real leak); a keystore layer needs native encryption on every platform and a migration of the user's saved servers, and only protects against a rooted or forensically imaged phone. To be done together with an optional app lock
- [x] Onboarding: on the first start with no servers, three skippable screens (what it does, how to add servers, what Android will ask) ending in the import sheet; not shown again, and not shown when servers exist. Widget-tested; **not seen on the phone** (it has servers, and clearing its data would delete them)
- [x] Profile sharing (QR/link) from the app: see Phase 1
- [x] Android backup is off (`allowBackup=false`): before, Auto Backup copied the app's data (every server's password and key) to the user's Google Drive. Verified on the device (`dumpsys package` no longer lists ALLOW_BACKUP)

### Phase 5 — More client protocols
- [x] OpenVPN via sing-box's `openvpn-client` endpoint: `.ovpn` import (inline certs, tls-auth/tls-crypt/tls-crypt-v2, multiple remotes, comp-lzo/compress, verify-x509-name), username/password prompt in the app, sing-box JSON import; end-to-end tests (TCP cert + tls-crypt, UDP user/pass + tls-auth); verified on device through omnitun-server
- [x] Hysteria v1, AnyTLS, SSH (password/key, host-key pinning), ShadowTLS v1-3 over Shadowsocks: links (`hysteria://`, `anytls://`, `ssh://`, `ss://…?plugin=shadow-tls`), Clash and sing-box JSON import; end-to-end tests; AnyTLS and ShadowTLS verified on device through omnitun-server
- [ ] OpenVPN XOR/Stunnel obfuscation — blocked upstream: sing-box 1.14.3's `openvpn-client` has no scramble/XOR option and there is no TLS-wrapping outbound to detour through; would need our own dialer
- [x] NordLynx: `nordvpn://<access token>?country=DE&limit=10` subscription → WireGuard profiles (key from `/v1/users/services/credentials`, servers from `/v1/servers/recommendations`); tested against a mock API (no NordVPN account to test live)
- [ ] Stealth (WireGuard over TLS/WS): sing-box WireGuard `detour` through a TLS/WS outbound with UDP-over-TCP (needs a sing-box server), later a wstunnel-compatible Go client
- [x] SSTP and IKEv2 in the core on pure-Go `veepin` (patched copy in `third_party/veepin`, see PATCHES.md: app-supplied TUN; EAP with a certificate-authenticated server). `sstp://` and `ikev2://` links (IKEv2: PSK, EAP-MSCHAPv2, certificates, IKE over TCP/RFC 8229). The app's own sockets stay off the tunnel (`exclude_self`). Tests: in-process veepin servers (SSTP, IKEv2 PSK + EAP over TCP) with a ping through the tunnel; live SSTP handshake + ping against VPN Gate; on device: SSTP to a dev server (`tools/sstpdev`) with ping through the tunnel; VPN Gate SSTP connects but that public server routes replies to a stale session
- [x] IKEv2 on the OS client (`system=1`, Android 11+ `Ikev2VpnProfile`) as an option — note many phones (e.g. Samsung A30s) lack `FEATURE_IPSEC_TUNNELS`, which is why the in-core engine is the default
- [x] IKEv2 with a certificate-authenticated server + EAP: the fork signs/verifies the server AUTH with its certificate in the EAP flow; tested client-to-server with our own server (pinned and wrong certificates). **Not yet tested against a real strongSwan/Windows/iOS peer**
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
- [x] App UI: "Bridge settings" on a Tor server is the profile editor's Tor form (none / Snowflake / obfs4 / your own bridge lines with Paste, File and Scan QR); the Import sheet keeps its "Tor" quick-add menu. Custom bridge lines are **not tested against the Tor network**
- [x] Tor exit address shown after connecting (Settings → "Show exit address and country", see Phase 7; verified on the A30s through Snowflake). The exit **country is not shown for Tor**: the lookup service answers `T1` for Tor exits (a Tor circuit has no fixed country anyway)
- [x] Snowflake's verbose logging: the transports write to the standard logger (every proxy, broker round trip and ICE step, naming the broker and its front domain); the core now points that logger at a filter (`core/tor/ptlog.go`) that drops routine lines and passes only lines that look like problems to the app log. Unit-tested; **not compared on the device**
- [ ] Desktop (Windows): run `omnitun-tor` next to the service; Tor on other ABIs (armeabi-v7a, x86_64) is built by the release script but untested

### Phase 7 — UI redesign (Mullvad-style, very simple)
- [x] Home like Mullvad VPN: dark and calm (`app/lib/ui/screens/home_screen.dart`): a lock in a ring (the real 0–100 % progress while connecting, frozen red at the failed step), the honest status, and once connected the **country** (flag + name), the server, timer and speed; at the bottom a location row and one big Connect / Cancel / Disconnect button. Light and "follow the system" themes are in Settings
- [x] **Dark-fantasy redesign (2026-10-17)**: cold muted-blue palette (`ui/theme.dart`), Cinzel/Inter/Vazirmatn fonts, no big ring in the middle (the globe is the picture; progress is a slim bar with the percent), a still starfield, small elements with all text under the globe, grouped rounded cards with icons in Settings, shorter texts everywhere (33 hints rewritten in all five languages), and **"Connected to <Country>"** with the flag, server and exit address (Tor says "Tor network"). **New app icon**: an arcane portal gate in a midnight starfield (Android adaptive icon with a themed monochrome layer, legacy and round PNGs, Windows `.ico`; `scripts/gen-icon.py`)
- [x] **Earth backdrop, Mullvad-style** (`ui/widgets/earth_globe.dart`, land from Natural Earth, public domain, `scripts/gen-globe.py`): a dotted globe (9 787 land cells) above the status text, nothing written on it. **Not connected:** it is turned to **where you really are** (red marker, the country under it as a chip, headline in red). **Connecting:** it flies to the country of the server about to be used (gold marker). **Connected:** it settles on the exit country the service saw (teal marker; the "you are here" marker stays, dimmer) with a dotted arc between the two; on disconnect it flies back. The flight takes 2 s with a zoom-out in the middle and is the only animation, so the globe costs no battery while still. **Where you are** comes from a direct request while *not* connected (Cloudflare's trace first, `api.country.is` as the fallback, because some networks block one of them), remembered for the next start; it follows the setting **Show exit address and country**, which is now **on by default** (it was off) and says it also looks while not connected; switched off, the globe rests and nothing is requested. The exit address on the home screen is shown in part (`147.78.x.x`, `2a07:d883:…`). A position for every country the app can name (test; small states approximate)
- [x] **Android release pipeline** (`.github/workflows/android.yml`, Android only; no other platform is built for now, details in `docs/release.md`): **a tag** (`0.1.0`, `v0.1.0`, `0.2.0-rc1`) builds the Tor client for three ABIs, the Go core with gomobile and the Flutter app as one release APK per ABI, **signed with the release key** (repository secrets; the release fails rather than falling back to a debug key), checks the signatures with `apksigner`, and publishes a GitHub Release with the APKs and `SHA256SUMS` (`gh release create`, `-` in the tag = pre-release). The Android version code is derived from the tag (0.1.0 → 100). **No tag** (push to main, pull request): only `flutter analyze`. `ci.yml` stays the test workflow. Third-party actions are pinned to commit SHAs (Dependabot updates them). The key: `scripts/android-keystore.sh` makes a PKCS12 keystore with a random 32-character password into the git-ignored `.env` (and `app/android/key.properties` for local builds); `scripts/android-github-secrets.sh` uploads it with `gh secret set`. Verified locally: a release built with `--build-name 0.1.0 --build-number 100` is signed by the new key (SHA-256 `8332…3820`); the history holds no keystore or password. **First tag run, `0.1.0` (2026-10-17):** the first attempt failed at the Go core: the runner's NDK 27 cannot link the Cronet static library of the NaiveProxy outbound (`unknown relocation (315)`), which the NDK 29 used locally can; the workflow now installs NDK 29 (`29.0.14206865`). The second run succeeded end to end (Tor client for 3 ABIs, gomobile core, app, signature check, release) and published https://github.com/NoobforAl/omnitun/releases/tag/0.1.0 with `omnitun-0.1.0-arm64-v8a.apk`, `-armeabi-v7a.apk`, `-x86_64.apk` (44-48 MB each) and `SHA256SUMS`. **Checked after downloading the release:** the checksums match, all three APKs are signed by the new key (SHA-256 `8332…3820`, the same as the keystore), package `dev.omnitun.app`, versionName `0.1.0`, versionCode 2100 (arm64: ABI offset 2000 + 100). A push to `main` ran only the lint job, which passed. **Not done:** the release APK has not been installed or run (it is signed with a different key from the app now on the A30s, so that phone would have to uninstall first, which deletes its data); the release is not marked as a pre-release although it is 0.x
- [x] **The Connect button is the progress** (`home_screen.dart`, `_MainButton`): while connecting it fills from the start side with the core's real progress and reads "Cancel · 48%"; tapping cancels. The step text stays under the globe. The globe and the status block form one group centred between the top bar and the bottom panel (a little above the middle) and shrink on short screens. Seen on the A30s (Tor connection)
- [x] **Open files and links with Omnitun** (`state/incoming.dart`, `ui/incoming_import.dart`, `MainActivity.kt`, `AndroidManifest.xml`): the OS offers Omnitun for share links (`vless://`, `vmess://`, `trojan://`, `ss://`, `hysteria2://`, `hy2://`, `hysteria://`, `tuic://`, `wireguard://`, `awg://`, `anytls://`, `sstp://`, `ikev2://`, `nordvpn://`, `naive+https://`), for OpenVPN (`application/x-openvpn-profile`) and WireGuard (`application/x-wireguard-profile`) profiles, for `.ovpn`, `.conf`, `.json`, `.yaml`, `.yml`, `.txt` files by name when the type is generic, and for "Share" of text or those files. An http(s) link opens the subscription form; everything else goes through the importer and is saved ("Imported 1 profile"). Nothing else is claimed: images and ordinary web links do not resolve to Omnitun. **Verified on the A30s:** `am start` with a `trojan://` link imported it; a text file opened through a real `content://` URI imported its two servers; `cmd package query-activities` resolves Omnitun for the OpenVPN/WireGuard types, `.ovpn`/`.conf`/`.yaml` names, `vless://` and SEND text, and not for images or `https://`. **Declared but not run:** iOS and macOS (`Info.plist` URL schemes, document types and `.ovpn`/`.conf` type declarations; the Swift side that receives them is **not written**), Linux (`linux/packaging/omnitun.desktop`, installed by CMake), Windows (`windows/packaging/omnitun-file-associations.reg`: adds Omnitun to "Open with" for the file types and claims the link schemes for the current user; run by hand, after editing the path). On Windows, Linux and macOS the system starts the app with the file or link as an argument, and the app reads it (`IncomingShare.fromArguments`, unit-tested with links, files, `file://`, flags, missing and oversized files; not run on those systems). Not done: a file chosen by extension on Android is matched by name only when the file manager passes a name; most pass the type (which is why the type filters exist)
- [x] **Offline country database (optional)** (`state/ipdb.dart`, `state/ipdb_state.dart`, `ui/screens/ipdb_screen.dart`; Settings → "Offline country database"): not installed by default. The user downloads the CC0 "geo-whois-asn-country" data (regional internet registries + ASN announcements, via the ip-location-db project on npm/jsDelivr; 22.9 MB as two CSV files), the app compiles it into a 6.9 MB file of 551 k merged ranges (IPv4 exact, IPv6 by the upper 64 bits; binary search), and uses it **to give a country to a server from its address** (a name is resolved first; known before connecting, no website asked) and **to give a country to an address** a lookup returned (the exit address, and where you are), taking precedence over the service's answer. Order of country sources: set by hand → seen when connected → database → guess from the name. A switch turns the use off without deleting it; Delete removes everything. **The downloader** resumes from the exact byte (`Range`), keeps the part file when it fails, is paused, or the app closes, tries four mirrors (fastly, unpkg, cdn, gcore), **checks each answer against the file's expected size and the finished file against its SHA-256** (from the jsDelivr data API; the checked values of the pinned version are built in for when the API is unreachable), skips a mirror that disagrees, and restarts a file whose checksum fails after a resume. The screen is a ring whose rune ticks light up as the bytes arrive, with the steps (IPv4, IPv6, Prepare), size, speed and time left; a failure turns it red, says where it stopped and offers Continue. Tests: the format and lookups (IPv4, IPv6 incl. the top bit, IPv4-in-IPv6, cut files), a local server that supports ranges and can drop the connection half-way or lie about sizes or compress (resume at the right byte, mirror fallback, stop and continue, wrong size, wrong bytes), the controller end to end (download → build → a server's country → switch off → delete), and the screen's states. **Verified on the A30s:** the full download over the phone's network; the mobile data and Wi-Fi switched off half-way (failed at 30 %, red, "Continue picks up right there"), switched on again, Continue went on at 44 % within seconds instead of starting over, both files passed their checksums, the database was built and the screen showed 551 k ranges · 6.9 MB; with it two servers with host names (VPNGate) moved from "Other" to Croatia and Japan; deleting it returned the screen to its first state. **A real bug the device found and the tests now guard:** Dart's client asks for gzip and `cdn.jsdelivr.net` / `gcore.jsdelivr.net` then report the size (and answer ranges) of a smaller compressed copy (416 for ranges past ~2 MB), so the total showed 3.1 MB and resuming would have corrupted the file; the client now asks for `identity`, and no mirror is trusted without the size and checksum. **Not done:** the database is not updated automatically (Update downloads the newest data again); the exit address / your own address still needs one request to a website — the database only supplies the country from it (no way to learn a public address without asking something); the strings of the screen are translated into the five languages by me and have not been read by native speakers
- [x] **No footer / navigation bar** (and no side rail): locations, logs and settings open from the home screen and close with Back. The bottom panel sits in the Scaffold's footer slot only so that toasts (the failover message) appear above it instead of over the button
- [x] Location picker like Mullvad (`location_screen.dart`): "Automatic" (fastest known server, switches on failure) on top, then countries with a flag, server count and latency dot; tapping a country connects to any server in it, the arrow opens its servers; search (from 7 servers on), test-speed button, "+" menu (import, subscription, manage servers); Tor has its own row; choosing a place while connected moves the connection there. Subscriptions and ordering stay in "Manage servers" (Settings or the "+" menu). *No city level*: profiles are servers, not cities
- [x] Country for every profile: set by hand ("Set country" in the server menu, remembered), else guessed from the name (flag emoji, `DE-Berlin-01`, `[US]`, country and big-city names) and the address (`de1.example.com`, `.nl`; generic domains like `.io` are ignored); the rest are under "Other". Unit-tested
- [x] Country from a lookup **through the tunnel** (opt-in, off by default: Settings → "Show exit address and country"): after connecting, the core asks `https://api.country.is/` through the running sing-box tunnel (`exit_info`, `Engine.TunnelDialer`; refuses when no tunnel can carry it, so it never describes the device's own address) and the home screen shows the exit address and the real country; the country is remembered for that server (so the location list files it under the right country, ahead of the name guess; a country set by hand still wins; never learned for Tor). Tested in the core (parsing of the common services' answers, an end-to-end lookup through an in-process tunnel, refusal without a tunnel), in the app and against the real daemon. **Verified on the A30s:** a Trojan server on the PC (reached over `adb reverse`, added through the app from its subscription URL) → "Connected · Germany · IP 147.78.241.51", the server moved into the Germany group, Tor showed its exit address. The service sees the exit address only (the request goes through the tunnel). No offline GeoIP database
- [x] Advanced things (split tunneling, DNS, bypass, kill switch, logs) are inside Settings; Protocol/profile editor screens are still to do (see Phase 1 and the GUI step)
- [x] Same screen on desktop and Android: one column up to 480 px wide, centered in wide windows (also the layout of a narrow "mini" window). A tray popup belongs to the Windows phase
- [ ] On-device check of the dark-fantasy screens: **seen on the A30s (2026-10-17, real phone, real server on the PC over `adb reverse`):** the home screen disconnected (Iran, red marker, found by the direct lookup), connecting (the flight, gold marker, dotted arc), connected ("Connected to Germany", teal marker, real exit `147.78.x.x`), in dark and light and with the smaller layout; Settings (restyled, Appearance row fixed) in dark and light; the location list; the bypass screen and list editor; the subscription form; real browser traffic through the tunnel (exit Germany); a bypass list with an address (`1.1.1.1`) went direct while the VPN stayed up (Cloudflare reported the phone's own Iranian address); in "only listed" mode with addresses only the listed address still went through the VPN and the unlisted one did not (not conclusive: the unlisted test address is also unreachable directly from this network). **Not seen on the phone:** the new launcher icon on the home screen and as a themed icon, Persian/Arabic with the new fonts, name-based bypass entries (sniffing path), battery numbers. Also seen on the A30s the same night: the Connect button as progress, opening a link and a file (above), and the offline database (above). Older note, still true for the older screens:

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
- [x] **Web API on huma + chi** (OpenAPI 3 generated, validation, `/docs`): `/api/v1/server`, `/protocols`, `/users`, `/users/{name}` (get, patch, delete), `/users/{name}/reset`, `/users/{name}/links`, `/users/{name}/ovpn/{proto}`, `/reload`; bearer token (`omnitun-server api token`), listens on 127.0.0.1 by default; started by `run`. **TLS for the API itself is not built in yet** (use an SSH tunnel or a reverse proxy; the server warns when it listens publicly)
- [x] **TUI on Bubble Tea** (+ bubbles, huh form for adding users): same features through the manager, tabs for Users / Protocols / Server, API token view; per-user traffic and logs come with Step 2
- [x] Tests for the manager, the API (httptest, incl. live token rotation) and the TUI; CLI checked by hand (exit codes 2/3/4 for invalid/not found/conflict)
- [x] CLI tests (cobra command tree against a temp data dir)

Step 2 — accounting and safety
- [x] Per-user traffic accounting and quota **enforcement**: sing-box connection counters summed per user (`server/stats`, `stats.json`), monthly quota (UTC calendar month, GB = 2^30 bytes) blocks the user until the month rolls over, the quota is raised or `user usage <name> --reset`; shown in CLI (`user list`, `user usage`), TUI, API (`usage` on users, `POST /users/{name}/usage/reset`) and in the subscription `Subscription-Userinfo` header; covered for VLESS, Trojan, Shadowsocks/ShadowTLS, Hysteria2, TUIC, AnyTLS, Naive and WireGuard (**not OpenVPN**: its endpoint does not report the user)
- [x] File locking (`state.json.lock`, works across processes): CLI, TUI, API and the running server can write at the same time without losing changes
- [x] Backup and restore: `omnitun-server backup <file>` / `restore <file> [--force]` (tar.gz of state, keys and certificates; hostile archives rejected) and `GET /api/v1/backup`
- [x] Live status: running/not running, connections and speed per user (CLI `status`/`user list`, TUI, API `server`)
- [x] Traffic per protocol (month, all-time, live connections: `proto list`, API `/protocols`, TUI) and the server log (`server.log`, rotated at 4 MB: `omnitun-server logs [-n N] [-f]`, `GET /api/v1/logs`, TUI Logs tab; the server's own messages and sing-box warnings/errors, so quota blocks, reloads and failures are visible remotely)
- [x] WireGuard per-user accounting: the endpoint reports no user, so traffic is attributed by the user's fixed tunnel address (verified end to end)
- [x] Per-user and per-protocol traffic of SSTP, IKEv2 and AmneziaWG: each host server implements `hostproto.Meter` (cumulative bytes per user since it started) and the accounting loop adds the changes every 2 s, so quotas, `user usage`, `proto list`, the API and the TUI cover them like the sing-box protocols. SSTP counts in the (patched) veepin server per user; AmneziaWG reads the device's per-peer counters (peer key to user); IKEv2 sums the live tunnels by their EAP user name (veepin reports the IKE ID, not the user, so `Peers()` now carries the EAP identity) and keeps the totals when a tunnel ends. Tested with the in-process servers (a ping is counted for the right user and stays counted after the client leaves) and with a fake meter in the runtime. **Limits:** IKEv2 can miss the traffic of the last 2 s before a client disconnects. Since 2026-10-17 these servers also run with real kernel TUN devices in the Docker test (see the status board), and report live sessions per user (SSTP: connected clients; IKEv2: live tunnels; AmneziaWG: a handshake in the last 3 minutes) from which the accounting loop derives connections and speed
- [x] OpenVPN per-user accounting: sing-box's OpenVPN server reports neither the user nor an address, so `third_party/sing-openvpn` (a patched copy, `replace` in `go.mod`) reports which user name holds which tunnel address (`SessionObserver`), and the collector charges connections from that address to that user, like WireGuard. **Verified in Docker with the real `openvpn` client (UDP and TCP):** a 2 MB download repeated for 24 s was counted for the user and the protocol alike, and the live connection and speed showed. The patch must be re-applied on upgrades of sing-openvpn
- [x] Live connections and speed for every protocol: the host servers report sessions per user and the runtime derives speed from the byte counters; the accounting loop publishes sing-box's and the host servers' numbers together (`publishLive`), where the collector used to replace everyone's. Unit-tested and verified in Docker for all six protocols
- [x] `make docker-e2e` (`tools/dockertest/run.py`): the end-to-end test described in the status board; `PROTO="sstp ikev2"` picks protocols, `--quick` only checks the tunnel

Step 3 — more protocols and setup
- [x] **Shared protocol registry** (`core/protocols`) used by both the server and the client: one definition per protocol with its `Spec` (name, label, transport, default port, options, needs-certificate, share-link), the server side (`Inbounds` as sing-box config) and the client side (`Profile` for a user); `core/links` exports any profile as a share link (and is what the app will use for share/QR). `serverconf` is now only glue; protocol keys, default ports, transports and the API's protocol metadata all come from the registry. A golden test pins the output: the sing-box server config, client profiles and `.ovpn` files are byte-identical to before the refactor, and share links keep their exact strings. **Adding a protocol = one type in `core/protocols/servers.go`** (+ an importer/engine mapping if the client needs new fields)
- [x] Client-side protocol form schema (profile editor) — `core/profile/form.go`, see Phase 1
- [x] Registry `Option`s driving the wizard and a server-creation form: **dropped for now**. Only one protocol has an option (the Reality site, which the wizard and `init --reality-site` already ask); it makes sense when a second option appears
- [x] ACME certificates: `omnitun-server cert acme --domain D [--email E] [--provider letsencrypt|zerossl|URL] [--dns cloudflare|alidns --dns-token …] [--http-port N]` / `cert status` / `cert self-signed` (also `GET /api/v1/certificate`, `PUT|DELETE /certificate/acme`). Uses sing-box's built-in ACME certificate provider (build tag `with_acme`, now in `scripts/go.ps1` and the Makefile): the VPN core obtains and renews the certificate live, inbounds reference the provider, the domain becomes the public address, links drop `insecure=1` and NaiveProxy needs no pinning. The subscription endpoint serves HTTPS with the ACME certificate (self-signed until the first issuance) and `api tls on` does the same for the web API (`server/tlsfiles`). DNS secrets are never returned by the CLI/API. Tested: generated config accepted by sing-box, link/profile changes, validation, HTTPS endpoints, certificate pick-up and renewal. **Not run against a real CA** (needs a public domain and open ports): try it on a real server before relying on it
- [x] Security: imported and rendered secrets are restricted to safe characters (no line breaks or semicolons in `.ovpn` files and ShadowTLS options)
- [x] **WireGuard server** (registry entry `wireguard`, sing-box WireGuard endpoint in userspace, no root needed): every user gets their own key pair and tunnel address (10.30.0.0/16; keys and addresses are added to existing servers and users on load; `user reset` rotates the keys and keeps the address), a `wireguard://` share link and a standard `.conf` download (`link <name> --file wireguard`, `/<token>/wireguard.conf`, `GET /users/{name}/files/wireguard`) for the official WireGuard apps; config files are now generic for every protocol that has one (OpenVPN `.ovpn` too). Verified: the generated client profile connects to the generated server and reaches a web server through the tunnel
- [x] AmneziaWG server: done as a host-level server, see the SSTP/IKEv2/AmneziaWG entry below
- [ ] ~~Tor bridge (obfs4/WebTunnel) server~~ dropped: a Tor bridge is a relay for the Tor network, not a VPN for the server's own users
- [x] **SSTP, IKEv2/IPsec and AmneziaWG servers** (registry protocols `sstp`, `ikev2`, `amneziawg`; off by default): served outside sing-box by `server/hostproto` and started by `run`. SSTP: TLS with the server/ACME certificate, MS-CHAPv2 users, port 8448. IKEv2: the server proves itself with its **certificate** (no secret shared between users; the vendored veepin server signs its AUTH with the certificate in the EAP flow now), users log in with EAP-MSCHAPv2, UDP 500/4500 plus IKE/ESP over TCP 4500; Omnitun profiles pin a self-signed certificate (`ca=` in the link), native Windows/iOS clients need a real certificate (`cert acme`). AmneziaWG (amneziawg-go): the users' WireGuard keys, addresses moved into 10.31.0.0/16, random per-server obfuscation parameters (valid by AmneziaWG's rules), `awg://` links and `.conf` downloads, UDP 51821. They need Linux and `CAP_NET_ADMIN` (the installer grants it and enables IP forwarding). NAT is set up and removed by the server (nftables, iptables fallback) and is narrow: clients may reach the internet, replies come back, and client-to-client and the host's private networks are dropped. Tested with fake TUN devices: the registry's generated profiles connect to the generated servers, wrong passwords/keys/pinned certificates are refused, NAT commands and the install plan are checked with a fake runner. **The real kernel TUN + NAT path was not run on a Linux host**: test on a VPS before use. Limits: amneziawg-go keeps the header values in package globals, so only one AWG device per process (fine for the server); Docker's FORWARD DROP policy can still block forwarding
- [x] Accounting no longer loses the last bytes of a connection: sing-box empties its list of closed connections at every garbage collection (it hooks the `sync.Pool` cleanup), so the collector now counts connections as they close from the manager's close events and keeps the list only as a fallback; deduplicated by connection ID; shutdown counts the connections it closes. Found through a test that failed about 1 run in 5 (also before this session); regression test forces GCs and has to count the traffic anyway. `Tracker.Save` is serialised (two concurrent saves shared one temp file)
- [x] Easy setup: `omnitun-server init` is a wizard in a terminal (public address auto-detected, protocols from the registry, ports, imitated site, first user) and fully scriptable with `--host --protocols a,b --port p=N --first-user`; choices are validated before anything is written and port clashes (same port, overlapping TCP/UDP) are rejected on init and on every port or enable change
- [x] `omnitun-server import xray <config.json> [--host H] [--adopt-reality]` and `POST /api/v1/import/xray`: moves an Xray/V2Ray server's users with their UUIDs and Trojan passwords, and (new server) the VLESS+Reality key, short ID, site and port, so users' existing links keep working
- [x] `omnitun-server install` / `uninstall` / `firewall` (Linux, root; `--dry-run` shows every step anywhere): system user, locked-down data directory, binary in `/usr/local/bin`, hardened systemd unit (also what `service` prints) started now and at boot, and the ports of the enabled protocols, the subscription endpoint, the ACME challenge and a public web API opened in **ufw or firewalld** (auto-detected; for plain nftables/iptables it prints the exact rules instead of guessing). The port list comes from the configuration (`FirewallPorts`), so changing protocols and running `firewall` again keeps the firewall right. The plan, commands and unit are unit-tested with a fake runner; **not run on a real Linux host yet**: try `install --dry-run` first, then on a fresh VPS
- [x] NAT/forwarding setup for the host-level servers (AmneziaWG, IKEv2, SSTP): see the entry above (nftables, iptables fallback; not run on a real host)

### Phase 9 — Polish & release
- [x] Translations (English, Persian, Russian, Chinese, Arabic) with right-to-left layout: `app/lib/l10n/app_*.arb` (about 390 messages each, generated classes next to them), a **Language** choice in Settings (system default or one of the five, each named in its own language; applies at once and is remembered), correct plural forms (Russian one/few/many, Arabic zero…many), Persian and Arabic mirror the layout (chevrons, back arrow, lists), addresses and examples inside right-to-left text keep their own order. The core's profile-editor labels and help texts are translated by the app (`l10n/form_text.dart`, keyed by the core's English text; a Dart test checks every label and help of the real core is covered). Tests: all languages have all messages with the same placeholders, plural rules, a smoke run of home, locations, add-server, settings and bypass in each language on a 360×760 phone (no overflow), the Settings language switch. Seen on the A30s in Persian (RTL, Samsung's own Persian font). **The core's own messages are translated too (2026-10-17):** editor problems ("Password is required"…), import and subscription failures, bypass-list line problems and Tor's progress steps now come with a code and values (`control.Error.Args`, `BypassIssue.Code`, `Progress.DetailCode`); the app translates them (`l10n/errors.dart`) and shows the core's English text for a code it does not know. Country names follow the language of the app (CLDR, `scripts/gen-country-names.py` → `state/country_names.dart`; the search matches English and local names). Tested: every code has a translation in all five languages, the real daemon's codes reach the app with their values, every country has a name. **Limits:** low-level network errors (the part after "download subscription:"), server and subscription names stay as they are; the translations were written by me and should be read by native speakers; Chinese and Arabic have not been looked at on a device
- [x] Open-source licenses inside the app: Settings → "Open-source licenses" lists every license text with the packages that use it (693 packages: Go, Dart, Rust), read from `app/assets/third_party_notices.md`, which `make notices` regenerates (`scripts/gen-notices.py`; run it after changing dependencies). Tested: the bundled file parses and covers Go, Dart and Rust; the screen opens a license; the failure text is translated
- [x] Developer commands: `make dev-server`, `dev-tui`, `dev-web` (a local server in `.dev/server`), `dev-android` (builds the core for the phone and runs the app), `fix` (gofmt, go fix, go vet, dart fix, cargo fmt), `docker-e2e`, `notices`, `help`. `make fix` ran clean on this tree
- [ ] Routing rules (geoip/geosite, bypass LAN, per-domain), IPv6 toggle, MTU
- [x] Local networks bypass the VPN by default (Settings → "Bypass local network"): 10/8, 172.16/12, 192.168/16, link-local, multicast, broadcast and IPv6 ULA/link-local are left out of the VPN routes for every engine (sing-box `route_exclude_address`, AmneziaWG/SSTP/IKEv2 route ranges) and sent direct; a tunnel's own subnet, DNS and explicitly routed WireGuard subnets stay inside; verified on device (`ip route` on tun0)
- [x] Bypass lists (a .txt file, pasted text, a link, or typed): **Settings → Bypass lists**. Entries, one per line: `example.com` (and its subdomains), `domain:`, `full:exact.example.com`, `keyword:bank`, `regexp:…`, `1.2.3.4`, `10.0.0.0/8`, `2001:db8::/32`; `#` comments; several named lists, each on/off; "only the listed destinations use the VPN" inverts it. The core (`core/singbox/bypasslist.go`) parses and checks the text (the app shows the count and the lines it will ignore, `geosite:`/`geoip:` are refused with a hint) and builds sing-box rules: one route rule per kind of entry (direct, or the VPN in "only these" mode), matching DNS rules so listed names are resolved the same way they are reached, and the listed address ranges are also left out of the TUN's routes. Tested: parser, sing-box accepts every shape of config (proxy and TUN, both modes), a real routing test (with a dead VPN a request only works when the list sends it direct; "only these" reverses it), control protocol (`parse_bypass`, `start` with `bypass`), the app's screens and a Dart test against the real daemon. **On the A30s:** the list is checked by the core as it is typed, and with Tor connected a listed site (`ipify.org`) that had worked through Tor stopped using Tor while Tor kept working (the direct path from this network cannot reach it, so a "real IP shown" result was not obtained); "only these", address ranges and the Android route exclusion were not tried on the device. No `geosite`/`geoip` databases (not bundled); 50 000 entries at most. **Battery and CPU (2026-10-17, `core/singbox/bypassopt.go`):** a list is optimized before use (names under a listed domain, names containing a listed keyword, nested and neighbouring ranges are merged); rules run cheapest first (DNS by port 53, then address rules, and only then packet sniffing, restricted to TLS/HTTP/QUIC and used **only when the list has names**); listed ranges are left out of the VPN's routes up to a budget of 512 routes (the biggest first; before, a 50 000-range list could produce tens of thousands of routes on Android); in "only these" mode with addresses only the VPN gets just those routes (`route_address`), so other traffic never enters the tunnel. Unit-tested (optimizer, route budget, rule order, no sniffing without names) and the engine's end-to-end bypass test passes; **not measured on the phone** (no battery numbers), and the new route handling was not exercised on the device with a list (only a Tor connection with the new rules was)
- [ ] Signed releases on GitHub (APK, Windows installer, server binaries for Linux amd64/arm64), F-Droid: the pieces exist (`scripts/build-android-release.sh` signs per-ABI APKs, `scripts/build-server-release.sh` builds the server for Linux and `omnitun-server --version` reports it, the notices generator), but nothing publishes a release; the Linux server binaries were built and checked as static ELF files, **never run**
- [x] User docs: `docs/app.md` (the app, location list, subscriptions, bypass lists, always-on VPN, messages) and `docs/server.md` (install, protocols and ports, users and quotas, certificates, API, backup, what is and is not tested), README refreshed

### Phase 10 — Peer-to-peer mesh VPN (planned, 2026-10-17; design in 2.4)
Layer 1 — join a mesh that already exists
- [ ] Spike (half a day): build sing-box with `with_tailscale` for Android and Windows, check the binary-size and battery cost, and connect to a Headscale instance run in Docker (`make docker-e2e` style: control server, two nodes behind two NAT routers)
- [ ] `tailscale://` profile (auth key, control URL, hostname, exit node, accept routes) in `core/profile`, importer, links, the form schema (editor) and `core/singbox` (endpoint); `exit_node` shows in the location list as a "My devices" group
- [ ] App: a "My devices" screen (peers, their addresses, online state, "use as exit"); MagicDNS names; works in the same Connect button flow (a mesh profile is a profile)
- [ ] Tests: config accepted by sing-box; engine end to end against Headscale in Docker (two clients in different NAT'd networks reach each other directly, then through a relay when direct paths are blocked); device check on the A30s
Layer 2 — our own coordinator (no third-party control server)
- [ ] Protocol (versioned, JSON over HTTPS): enroll with a one-time key from `omnitun-server mesh invite`, long-poll for the peer list, report my endpoints; keys are WireGuard keys, the server never sees private keys
- [ ] `server/mesh`: device registry per user, address allocation from a range of our own, peer maps, STUN endpoint (UDP 3478), relay (forwards encrypted WireGuard packets, accounted per user and capped); CLI `mesh invite|devices|revoke`, API and TUI like the other features; docs
- [ ] `core/mesh`: NAT-type detection, endpoint discovery, simultaneous-open hole punching, path upgrade from relay to direct, keepalives that are cheap on battery (Android doze), roaming between Wi-Fi and mobile
- [ ] Tests in Docker with NAT routers (full-cone, restricted, symmetric): direct path when possible, relay when not, a peer that changes network keeps the session
Layer 3 — exit through a friend
- [ ] Invite links, per-peer caps (speed, monthly bytes), allow/deny lists, a persistent notification while sharing (Android), revoke; the exit NATs and drops private ranges and other guests
- [ ] Abuse and legal text in the app and docs (the owner of the exit is the one the internet sees)
Cross-cutting
- [ ] Obfuscation: carry the mesh's WireGuard over AmneziaWG parameters when the network blocks plain WireGuard; the relay also reachable over TLS (TCP 443) for UDP-hostile networks
- [ ] Settings and onboarding copy in five languages; Windows later (Phase 3)

### Phase 11 — DNSCrypt (planned, 2026-10-17; design in 2.5)
- [ ] `core/dnscrypt`: DNS stamp parser (DNSCrypt, DoH, DoT, DoQ, relays), certificate fetch and verification, the local loopback forwarder (UDP + TCP, in-flight limits, small cache), Anonymized DNSCrypt through a relay, fail-closed option; sockets through the tunnel's outbound
- [ ] `Options.DNS` accepts `sdns://…`; `core/singbox` points the DNS server at the forwarder; routing rules for the resolver and relay addresses so DNS never leaks around the VPN; Tor profiles use TCP only
- [ ] Bundled signed resolver list (`scripts/gen-resolvers.py` downloads `public-resolvers.md`, checks the minisign signature, writes a Dart/Go table with country and properties); refreshed through the VPN
- [ ] App: Settings → DNS picker (system default, Cloudflare, Quad9, a few DNSCrypt resolvers, "Anonymized" switch, custom stamp), a "DNS: DNSCrypt · <resolver>" line in the connected details, a clear error when the stamp is invalid; strings in five languages
- [ ] Server (optional, off by default): DNSCrypt resolver for the server's users (`omnitun-server proto enable dnscrypt`, key generation and rotation, `firewall` opens the port, `link --stamp`)
- [ ] Tests: stamp parsing against the published examples; an in-process DNSCrypt server (the library's) answers through the forwarder; a wrong provider key is rejected; relay path; a leak test (with the VPN up, no plain DNS leaves the machine: capture in Docker with `tcpdump`); on the A30s a DNS leak check
- [ ] Docs: `docs/app.md` (what the DNS setting does and does not protect) and `docs/server.md`

### Later
- [ ] iOS / macOS (NetworkExtension), Linux desktop client

---

## 8. Risks and open questions

- **Lightway**: the Rust implementation is AGPL-3.0 and currently supports Linux/macOS only. Windows/Android clients need porting → lowest priority.
- **Arti on Android**: size and battery cost; PT executables must ship as `lib*.so` in `jniLibs` to be executable.
- **App stores**: Google Play's VpnService policy requires declaring VPN as core functionality; some protocols may be restricted in some regions' stores → also ship via GitHub Releases / F-Droid.
- **Windows code signing**: the driver (Wintun) is already signed; our service and installer need an EV/OV certificate for SmartScreen.
- **Binary size**: Go core with sing-box (all protocol tags) ≈ 30–50 MB; use build tags so add-ons stay optional. The Tailscale endpoint (Phase 10) adds a lot; measure first and keep it a build tag.
- **Mesh and censorship**: Tailscale's control servers and DERP relays are blocked in some countries; layer 1 therefore also supports Headscale, and layer 2 exists so a mesh can run on the user's own server. UDP hole punching fails behind symmetric NAT and CGNAT, which is common on mobile networks: a relay is needed, and it costs the server's bandwidth.
- **Mesh and battery**: keeping a path to peers alive on Android (doze, background limits) conflicts with the mobile-first goal; the mesh must be opt-in and idle cheaply.
- **Sharing an exit** makes the owner liable for what guests do; opt-in, caps and clear text are required before layer 3 ships. Google Play policy on VPN apps that relay other users' traffic needs checking.
- **DNSCrypt reachability**: its usual ports (443, 5353) and its resolver list (GitHub) may be blocked; always keep TCP/443 and a bundled list, and test from Iran before calling it done. Unlike DoH it is not disguised as web traffic.

---

**Dependency note (2026-10-16):** every Go module is at its latest release except the `pion/*` family, which is pinned to the versions Snowflake v2.15.1 builds against (`webrtc` v4.2.18, `ice` v4.4.1, `transport/v4` v4.1.0, …); newer pion releases move to `transport/v5` and break the Snowflake client. Lift the pin when Snowflake releases for it (`go list -m -u all` shows what is held). Do not run a blind `go get -u ./...`; after upgrades run `scripts/go.ps1 test ./...` and an Android vet (`GOOS=android go vet ./core/mobile`).

## 9. References

- DNSCrypt: https://www.dnscrypt.org/ · protocol v2 and DNS stamps: https://dnscrypt.info/protocol and https://dnscrypt.info/stamps-specifications · resolver list: https://github.com/DNSCrypt/dnscrypt-resolvers · Go library: https://github.com/ameshkov/dnscrypt · reference client/relay/server: https://github.com/DNSCrypt/dnscrypt-proxy
- Mesh VPNs: Tailscale's NAT traversal write-up https://tailscale.com/blog/how-nat-traversal-works · Headscale (self-hosted control server) https://github.com/juanfont/headscale · Nebula https://github.com/slackhq/nebula · sing-box `tailscale` endpoint https://sing-box.sagernet.org/configuration/endpoint/tailscale/

- Lightway (Rust, AGPL-3.0): https://github.com/expressvpn/lightway
- Arti / pluggable transports: https://blog-7551d3.pages.torproject.net/arti_110_released/ · https://docs.rs/tor-ptmgr
- Arti on Android: https://tordev.guardianproject.info/tor-on-android/arti-mobile-on-android
- hev-socks5-tunnel (MIT): https://github.com/heiher/hev-socks5-tunnel
- amneziawg-go (MIT): https://www.freshports.org/net/amneziawg-go
- TUIC crate (GPL-3.0, protocol license-free): https://rust-digger.code-maven.com/crates/tuic
- Hiddify (Flutter + sing-box, GPL-3.0): https://aur.archlinux.org/packages/hiddify
- Flutter cross-platform VPN case study: https://www.billingserv.com/blog/why-cross-platform-vpn-support-led-us-to-rebuild-our-apps-in-flutter/
- VPN UI patterns: https://nordvpn.com/blog/mobile-app-redesign-2026/ · https://nordvpn.com/zh/blog/desktop-app-redesign-2026/ · https://vp0.com/blogs/vpn-connection-switch-ui-clone-ios · https://0xacab.org/leap/bitmask-vpn/-/issues/513
