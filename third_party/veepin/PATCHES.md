# Omnitun patches to veepin

Upstream: https://github.com/xen0bit/veepin (MIT), v0.9.6 (2026-09-03).
Trimmed: docs, tests, CLI (`cmd/`), packaging and upstream agent files.

1. `dataplane/tun_opener*.go`, `dataplane/tun_linux.go`, `dataplane/tun_other.go`:
   `dataplane.TUNOpener` lets the embedding app supply the TUN device (Android's
   VpnService fd, or a test device) instead of `/dev/net/tun`.
2. `ikev2/client.go`, `ikev2/server.go`, `sstp/sstp.go`, `sstp/server.go`: open
   the TUN with `dataplane.OpenTUNFor`, passing the negotiated addresses so the
   app can build its interface after the handshake.
3. `internal/ikev2/ike/client.go`, `ikev2/client.go`: EAP-MSCHAPv2 without a PSK
   authenticates the server by its certificate (CERTREQ + optional IDr in the
   first IKE_AUTH, chain verified against `CAFile` or the system roots) — the
   usual strongSwan / provider setup. Upstream only supported EAP with a PSK.
4. `sstp/server.go`: `Server.Traffic()` counts the inner bytes each user moved
   (up = from the client, down = to the client), kept after their sessions end.
   `internal/ikev2/ike/peers.go`, `ikev2/server.go`: the EAP-MSCHAPv2 user name
   is reported as the peer's `Name` (the IKE ID is whatever the client sends), so
   `Peers()` can be summed per user.
