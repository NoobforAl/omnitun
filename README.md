# Omnitun

Cross-platform VPN client (Flutter) with a Go/Rust core, plus a CLI for running your own VPN server.
See [PLAN.md](PLAN.md) for the architecture and roadmap.

## Layout

| Path | What |
|---|---|
| `app/` | Flutter client (Android, Windows; iOS/macOS/Linux scaffolded) |
| `core/profile` | Protocol-neutral profile model |
| `core/importer` | Share links (`vless`, `vmess`, `trojan`, `ss`, `hy2`, `tuic`, `wireguard`), base64 subscriptions, WireGuard/AmneziaWG `.conf` |
| `core/singbox` | Profiles → sing-box config (proxy or TUN mode, selector + urltest groups) |
| `core/engine` | Runs sing-box in-process: start/stop, live profile switching, traffic stats, latency tests |
| `core/control` | UI ↔ core protocol (loopback TCP, JSON lines, token auth); tunnel stops when no UI is connected |
| `core/netcheck` | "Is there internet at all?" check, so we report *no internet* instead of failing over |
| `cmd/omnitun-svc` | Core daemon the desktop app launches (exits when the app dies) |
| `cmd/omnitun` | Developer CLI for the core |
| `tor/` | Rust Tor module (Arti) with a C ABI |

## Develop

```sh
# Go core — always build with the sing-box feature tags:
#   Windows: ./scripts/go.ps1 <cmd> ...     Linux/macOS: make test / make build
./scripts/go.ps1 test ./...
./scripts/go.ps1 run ./cmd/omnitun import links.txt
./scripts/go.ps1 run ./cmd/omnitun config links.txt
./scripts/go.ps1 run ./cmd/omnitun run links.txt   # SOCKS5/HTTP proxy on 127.0.0.1:2080

# Rust
cd tor && cargo test

# Flutter
cd app && flutter test && flutter run -d windows
```

On desktop the app launches `bin/omnitun-svc(.exe)` (or `OMNITUN_SVC`, or the copy next to the app executable), so build it first:
`./scripts/go.ps1 build -o bin/ ./cmd/...`. Without it — and on Android until the mobile bridge lands — the app falls back to a preview engine and says so in Logs.

## License

MIT
