# Omnitun server

`omnitun-server` is one program that serves many VPN protocols to your own users, with one subscription link per user, quotas and traffic accounting, a terminal UI and a web API.

**Honest status.** The sing-box protocols (the first ten below) are tested end to end against Omnitun's own client. SSTP, IKEv2/IPsec and AmneziaWG, the installer, the firewall helpers and ACME were written and unit-tested here on Windows with fake TUN devices and a fake command runner; **none of them has run on a real Linux host yet**. Try `install --dry-run` first, then a fresh VPS you can throw away.

## Quick start (a VPS you own)

```sh
# 1. put the binary on the server (a release build, or: go build ./cmd/omnitun-server)
# 2. as root:
omnitun-server init                    # a guided wizard: address, protocols, ports, the site to imitate, first user
omnitun-server install --dry-run       # shows every step
omnitun-server install                 # system user, hardened systemd unit, starts now and at boot, opens the firewall
omnitun-server link alice --qr         # links and QR codes for alice; --sub prints only her subscription URL
```

No terminal wizard? Give everything as flags:

```sh
omnitun-server init --host vpn.example.com --protocols vless-reality,trojan,hysteria2 \
    --port trojan=9443 --first-user alice
```

`init` checks your choices before writing anything; two protocols on the same port (including TCP/UDP overlaps that clash) are refused.

## Protocols

| Protocol | Default port | Notes |
|---|---|---|
| VLESS + Reality | 443/tcp | imitates a real site (`--reality-site`); needs no certificate |
| Shadowsocks 2022 | 8388/tcp+udp | |
| Trojan | 8443/tcp | TLS with the server certificate |
| Hysteria2 | 8443/udp | |
| TUIC v5 | 8444/udp | |
| AnyTLS | 8445/tcp | |
| ShadowTLS v3 | 8446/tcp | with Shadowsocks inside |
| NaiveProxy | 8447/tcp | needs a certificate clients trust (use ACME) |
| WireGuard | 51820/udp | every user gets a key pair and a fixed address (10.30.0.0/16); standard `.conf` download |
| OpenVPN UDP / TCP | 1194 | `.ovpn` download; per-protocol traffic only (the server cannot tell users apart) |
| SSTP | 8448/tcp | **Linux, root or `CAP_NET_ADMIN`**; off by default |
| IKEv2/IPsec | 4500/udp (+ 500/udp, TCP 4500) | **Linux, root or `CAP_NET_ADMIN`**; off by default; the server proves itself with its certificate, users sign in with EAP-MSCHAPv2 |
| AmneziaWG | 51821/udp | **Linux, root or `CAP_NET_ADMIN`**; off by default; random obfuscation parameters per server; AmneziaWG 1.x only |

```sh
omnitun-server proto list                  # ports, status, traffic this month
omnitun-server proto enable sstp
omnitun-server proto port trojan 9443
omnitun-server firewall                    # open the ports again after changing protocols
```

SSTP, IKEv2 and AmneziaWG need a TUN device and NAT on the host. The server sets up and removes narrow nftables (or iptables) rules itself: clients may reach the internet; client-to-client traffic and the host's private networks are dropped. Docker's own firewall policy can still block forwarding. Native Windows and iOS IKEv2/SSTP clients need a certificate they trust, so use `cert acme` for those.

## Users, links and quotas

```sh
omnitun-server user add alice --days 30 --quota 100 --note "Alice's phone"
omnitun-server user list                   # state, expiry, usage, live speed
omnitun-server user quota alice 200        # GB per month, 0 = none
omnitun-server user expire alice never
omnitun-server user disable alice
omnitun-server user reset alice            # new credentials; the old links stop working
omnitun-server link alice --ovpn udp       # the OpenVPN config file
```

A user's subscription URL carries the usage, quota and expiry in the standard `Subscription-Userinfo` header, so apps like Omnitun show them. The monthly quota is counted per UTC calendar month and **blocks** the user (their connections are dropped) until the month rolls over or you raise it. `user usage <name> --reset` clears the count.

Traffic is counted per user and per protocol for every protocol that can tell users apart. Not counted per user: OpenVPN. For SSTP, IKEv2 and AmneziaWG there is no live speed or connection count, and IKEv2 can miss the last two seconds before a client disconnects.

## Certificates

The server starts with a self-signed certificate (clients are told to pin it). For a real one:

```sh
omnitun-server cert acme --domain vpn.example.com --email you@example.com   # Let's Encrypt by default
omnitun-server cert status
omnitun-server cert self-signed            # go back
```

`--dns cloudflare|alidns --dns-token …` uses a DNS challenge; otherwise port 80 must be reachable (`--http-port` to change it). The certificate renews while the server runs. **ACME has not been run against a real CA from here**: try it on a real domain before relying on it.

## Web API, terminal UI, logs

```sh
omnitun-server api enable                  # then `run` serves it; docs at /docs (OpenAPI)
omnitun-server api token                   # the bearer token; --new replaces it at once
omnitun-server api tls on                  # HTTPS with the server certificate
omnitun-server tui                         # users, links with QR codes, protocols, server, logs
omnitun-server logs -f                     # what the server reported: reloads, quota blocks, warnings
```

The API listens on 127.0.0.1 by default. Reach it over an SSH tunnel, or turn on `api tls` before listening publicly (the server warns when it listens publicly without TLS).

## Backup, restore, moving from Xray

```sh
omnitun-server backup omnitun.tar.gz       # state, keys, certificates: it holds every secret
omnitun-server restore omnitun.tar.gz [--force]
omnitun-server import xray /etc/xray/config.json --host vpn.example.com --adopt-reality
```

`import xray` moves an Xray or V2Ray server's users (with their UUIDs and Trojan passwords) and, for a new server, the VLESS+Reality key, short id, site and port, so existing client links keep working.

## Files

The data directory is `/var/lib/omnitun-server` as root and `~/.omnitun-server` otherwise (`--data` or `OMNITUN_SERVER_DATA`). It holds `state.json` (users and keys), `stats.json`, `server.log` and the certificates. Several commands can write at the same time; changes are picked up by a running server without a restart.
