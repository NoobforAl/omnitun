# Omnitun app

One calm screen: an Earth that shows where you are and where you exit, the state in words, and one button.

## Look

A dark-fantasy look in cold, muted blues (a light "stone" theme is in Settings). Behind the home screen is a dotted Earth: when you connect it turns to the exit country and marks it with a glow; it moves only for that turn. Fonts are open (SIL OFL, texts in the app's assets): Cinzel for headings, Inter for text, Vazirmatn for Persian and Arabic. The land outlines are Natural Earth (public domain). `scripts/gen-icon.py`, `gen-fonts.py` and `gen-globe.py` rebuild the icon, the fonts and the globe data.

## First start

Three short screens say what the app does, how to add servers and what Android will ask. **Skip** is always there. They are only shown when the app has no servers yet.

## The home screen

* The ring fills while connecting (the real progress: network check, configuration, engine, tunnel, handshake). If something fails it stops red at the step that failed and says why, in your language.
* "Connected" is only shown after a real test connection through the tunnel worked.
* **Not connected:** the Earth is turned to where you really are, with a red marker and your country (found with a direct request to Cloudflare or `api.country.is`; Settings → "Show exit address and country" turns it off, and then the Earth just rests).
* **Connecting:** the Earth flies to the server's country and a slim bar shows the real progress. **Connected:** it says **Connected to <country>** with the flag, the server's name and part of the exit address (`147.78.x.x`). Disconnecting flies back. Nothing is written on the Earth.
* **Location** opens the list of places. **Connect / Cancel / Disconnect** is the only button.
* Settings (the sliders icon) is at the top, and Logs are in Settings. Everything else opens from there and closes with Back; there is no footer.
* A notification (Android) shows the speed and has a Disconnect button.

## Choosing where to connect

* **Automatic**: the fastest server you know (run the speed test, the gauge icon), switching to the next one if a server stops working.
* **A country**: any server in that country, fastest first. The arrow opens its servers; tap one to use only that server.
* **Tor**: has its own row; it only runs when you pick it.
* Choosing a place while connected moves the connection there.

Countries come from what you set ("Set country" in a server's menu or in its editor), else they are guessed from the name (a flag emoji, `DE-Berlin-01`, `[US]`, a country or big-city name) and from the address (`de1.example.com`, `.nl`). Servers whose country is not known are under **Other**. The app does not look servers up in a GeoIP database.

## Adding servers

**+** in the location list:

* **Import a config, link or QR**: share links (`vless://`, `trojan://`, `ss://`, `hy2://`, `tuic://`, `anytls://`, `ssh://`, `sstp://`, `ikev2://`, `wireguard://`, `awg://`, …), Clash and sing-box configs, Xray client JSON, WireGuard/AmneziaWG `.conf`, OpenVPN `.ovpn`, Tor bridge lines, or a subscription link. Paste, open a file or scan a QR code.
* **Add a server by hand**: pick the protocol; the form is checked by the same core that connects, and its messages are shown as they are.
* **Add a subscription**: URL, optional authentication (none, Basic, Bearer, a custom header, a token in the URL) and how to download it: directly, **directly or through the VPN if the provider is blocked**, or always through the VPN. Usage and expiry from the provider are shown, and it refreshes on the provider's interval.

A server's menu: **Edit**, **Share link or QR** (scan it on another device), **Set country**, **Delete**. Editing a server that came from a subscription is overwritten the next time the subscription refreshes. **Manage servers** (Settings, or the **+** menu) is where you reorder servers (failover order), manage subscriptions and delete.

For Tor, **Bridge settings** is the editor: no bridge, Snowflake, obfs4 or your own bridge lines (paste, file, QR). Where Tor is blocked, Snowflake worked on the test phone; the built-in obfs4 bridges did not.

## Bypass lists

Settings → **Bypass lists**: websites and addresses that should not use the VPN (your bank, local services). One entry per line:

```
example.com            the domain and its subdomains
full:www.example.com   exactly this name
keyword:bank           names containing this word
regexp:^ads?\..*       names matching an expression
203.0.113.7            an address
10.0.0.0/8             a range (IPv4 or IPv6)
# a comment
```

Add a list by typing, pasting, opening a `.txt` file or downloading a link. The app shows how many entries it understood and which lines it will ignore. Each list can be switched off. **Only the listed destinations use the VPN** turns it round: everything else goes direct. Changes apply the next time you connect.

**Battery and speed.** The app tidies a list before using it: a name under another listed domain, a name containing a listed keyword, and nested or neighbouring address ranges are merged, so the matcher stays small. Traffic to listed addresses is decided first and never inspected; names are only recognised (from the first packet of a connection) when your list has names in it. Listed address ranges are kept out of the VPN's routes (the largest first, up to a limit of a few hundred routes; the rest are still bypassed, one step later). In "only the listed" mode with only addresses, nothing else enters the VPN at all.

## Settings worth knowing

* **VPN mode** off = proxy only (a local SOCKS/HTTP port, no VPN permission). On Android it is on by default.
* **Auto-switch on failure**: try the next server when one stops working.
* **Bypass local network**: LAN devices stay reachable while connected.
* **Always-on VPN and kill switch** (Android): opens the system's VPN settings. Turn on "Always-on VPN" (Android reconnects at boot and after the app was stopped, using the last connection) and, if you want, "Block connections without VPN" (no traffic while the VPN is down). Connect once from the app first.
* **Split tunneling** (Android): choose which apps use the VPN, or which bypass it.
* **Remote DNS**: the resolver used through the tunnel (`https://…`, `tls://…`, `tcp://…` or an IP).
* **Language**: English, فارسی, Русский, 中文, العربية (the layout mirrors for Persian and Arabic). **Appearance**: dark, light or the system's.

## What the messages mean

| Message | Usually |
|---|---|
| The server did not answer in time | the server is down, or blocked from your network; try another protocol or server |
| The server refused the connection | nothing listens on that port, or a firewall rejected it |
| The server's security certificate was rejected | a self-signed certificate that was not pinned, or a wrong date on the phone |
| The username or password was rejected | edit the server and check them (OpenVPN, SSTP, IKEv2) |
| No internet connection | the device itself is offline; nothing is wrong with the server |
| Tor did not finish connecting | this network blocks Tor; use a Snowflake bridge |

## Privacy

Server passwords and keys are kept in the app's private storage; Android backup of the app is off, so they are not copied to a cloud account. Tor never logs the destinations it connects to. The log screen can be copied or cleared.
