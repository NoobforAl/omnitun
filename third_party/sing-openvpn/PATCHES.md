# Omnitun patches to sing-openvpn

Based on `github.com/sagernet/sing-openvpn@v0.1.1-0.20260929151220-7c3045dbddce`
(`.github/` removed). Used through `replace` in the root `go.mod`.

1. `server_tunnel.go`: `SessionObserver` / `SessionEvent` report which user name
   holds which tunnel address when a peer is assigned or loses its addresses.
   sing-box's OpenVPN server reports neither the user nor a stable address
   with the connections it routes, so the Omnitun server cannot charge traffic
   to a user without it.
