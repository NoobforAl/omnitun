#!/usr/bin/env python3
"""Docker end-to-end tests of the server's VPN protocols with real clients on real Linux.

A server container (this repo's Linux binary, with a TUN device and NAT) and, per
protocol, a fresh client container with the standard client for it: OpenVPN, WireGuard
(wg-quick), SSTP (sstp-client + pppd), IKEv2 (strongSwan) and AmneziaWG (our own
client on a kernel TUN). The client reaches a "web" container on a network only the
server can reach, through the tunnel; the answer shows the address the web server saw,
which proves the server's NAT. Then the server's traffic accounting is read.

    python tools/dockertest/run.py                  # every protocol
    python tools/dockertest/run.py sstp ikev2       # some of them
    python tools/dockertest/run.py --keep           # leave the containers running

Needs Docker, Go and Python 3. About 3 minutes the first time (images).
"""
import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.parse

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
BUILD = os.path.join(HERE, "build")
TAGS = "with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_openvpn,with_openconnect,with_acme,badlinkname,tfogo_checklinkname0"

NET_A, NET_B = "e2e-a", "e2e-b"           # clients + server / server + "internet"
SERVER_A, SERVER_B, WEB_B = "172.30.0.2", "198.18.0.2", "198.18.0.10"
PROTOCOLS = ["openvpn-udp", "openvpn-tcp", "wireguard", "amneziawg", "sstp", "ikev2"]
PROTO_KEY = {"openvpn-udp": "openvpn-udp", "openvpn-tcp": "openvpn-tcp", "wireguard": "wireguard",
             "amneziawg": "amneziawg", "sstp": "sstp", "ikev2": "ikev2"}


def sh(*cmd, check=True, timeout=300, quiet=False):
    r = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout)
    if check and r.returncode != 0:
        raise RuntimeError("%s\n%s%s" % (" ".join(cmd), r.stdout, r.stderr))
    return r


def dx(container, script, check=True, timeout=120):
    """Runs a shell script in a container."""
    return sh("docker", "exec", container, "sh", "-c", script, check=check, timeout=timeout)


def build():
    os.makedirs(BUILD, exist_ok=True)
    env = dict(os.environ, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0")
    print("building the Linux binaries...")
    subprocess.run(["go", "build", "-tags", TAGS, "-trimpath", "-ldflags", "-s -w -checklinkname=0",
                    "-o", os.path.join(BUILD, "omnitun-server"), "./cmd/omnitun-server"], cwd=ROOT, env=env, check=True)
    subprocess.run(["go", "build", "-trimpath", "-o", os.path.join(BUILD, "awgclient"), "./tools/dockertest/awgclient"],
                   cwd=ROOT, env=env, check=True)
    shutil.copy(os.path.join(HERE, "web.py"), os.path.join(BUILD, "web.py"))
    print("building the images...")
    for tag, dockerfile in (("omnitun-e2e-server", "Dockerfile.server"), ("omnitun-e2e-client", "Dockerfile.client"),
                            ("omnitun-e2e-web", "Dockerfile.web")):
        sh("docker", "build", "-q", "-t", tag, "-f", os.path.join(HERE, dockerfile), BUILD, timeout=1200)


def cleanup():
    for c in ("e2e-client", "e2e-server", "e2e-web"):
        sh("docker", "rm", "-f", c, check=False)
    for n in (NET_A, NET_B):
        sh("docker", "network", "rm", n, check=False)


def start_world():
    cleanup()
    sh("docker", "network", "create", "--subnet", "172.30.0.0/24", NET_A)
    sh("docker", "network", "create", "--subnet", "198.18.0.0/24", NET_B)
    sh("docker", "run", "-d", "--name", "e2e-web", "--network", NET_B, "--ip", WEB_B, "omnitun-e2e-web")
    sh("docker", "run", "-d", "--name", "e2e-server", "--network", NET_A, "--ip", SERVER_A,
       "--cap-add", "NET_ADMIN", "--device", "/dev/net/tun", "--sysctl", "net.ipv4.ip_forward=1",
       "omnitun-e2e-server", "sleep", "infinity")
    sh("docker", "network", "connect", "--ip", SERVER_B, NET_B, "e2e-server")
    protos = "openvpn-udp,openvpn-tcp,wireguard,amneziawg,sstp,ikev2"
    print(dx("e2e-server", "omnitun-server --data /data init --host %s --protocols %s --first-user alice" % (SERVER_A, protos)).stdout[-300:])
    sh("docker", "exec", "-d", "e2e-server", "sh", "-c", "omnitun-server --data /data run > /var/log/run.log 2>&1")
    for _ in range(40):
        time.sleep(1)
        log = dx("e2e-server", "cat /var/log/run.log", check=False).stdout
        if "Ctrl+C" in log:
            break
    print("--- server log ---")
    print(log.strip())
    print("------------------")


def user_info():
    out = dx("e2e-server", "omnitun-server --data /data link alice --json").stdout
    d = json.loads(out)
    urls = {l["protocol"]: l["url"] for l in d["links"]}
    sub = d["subscription"]
    password = urllib.parse.unquote(urllib.parse.urlparse(urls["sstp"]).password)
    return sub, password, d


def usage():
    r = dx("e2e-server", "omnitun-server --data /data proto list --json", check=False)
    try:
        return json.loads(r.stdout)
    except ValueError:
        return r.stdout


def start_client():
    """A client container. Privileged because SSTP's pppd needs /dev/ppp, which Docker's VM
    only creates for privileged containers."""
    sh("docker", "rm", "-f", "e2e-client", check=False)
    sh("docker", "run", "-d", "--name", "e2e-client", "--network", NET_A, "--privileged",
       "omnitun-e2e-client", "sleep", "infinity")


def wait_for(container, script, what, seconds=25):
    for _ in range(seconds):
        if dx(container, script, check=False).returncode == 0:
            return True
        time.sleep(1)
    print("  timed out waiting for", what)
    return False


def fetch(path, dest):
    return dx("e2e-client", "curl -sfk %s -o %s" % (path, dest))


# ------------------------------------------------------------------ the clients


def t_openvpn(proto):
    def run(sub, pw, _):
        fetch(sub + "/%s.ovpn" % proto, "/tmp/c.ovpn")
        dx("e2e-client", "printf 'alice\\n%s\\n' > /tmp/auth && chmod 600 /tmp/auth" % pw)
        dx("e2e-client", "openvpn --config /tmp/c.ovpn --auth-user-pass /tmp/auth --daemon --log /tmp/ovpn.log "
                         "--pull-filter ignore redirect-gateway --pull-filter ignore dhcp-option "
                         "--route 198.18.0.0 255.255.255.0 --verb 3")
        return wait_for("e2e-client", "grep -q 'Initialization Sequence Completed' /tmp/ovpn.log", "OpenVPN to come up", 30)
    return run


def t_wireguard(sub, pw, _):
    fetch(sub + "/wireguard.conf", "/tmp/wg0.conf")
    dx("e2e-client", "sed -i '/^DNS/d;s#^AllowedIPs.*#AllowedIPs = 198.18.0.0/24#' /tmp/wg0.conf")
    r = dx("e2e-client", "wg-quick up /tmp/wg0.conf 2>&1", check=False)
    if r.returncode != 0:
        print(r.stdout)
        return False
    return True


def t_amneziawg(sub, pw, _):
    fetch(sub + "/amneziawg.conf", "/tmp/awg.conf")
    sh("docker", "exec", "-d", "e2e-client", "sh", "-c", "awgclient -conf /tmp/awg.conf -route 198.18.0.0/24 > /tmp/awg.log 2>&1")
    return wait_for("e2e-client", "grep -q 'awgclient: up' /tmp/awg.log", "the AmneziaWG client", 15)


def t_sstp(sub, pw, _):
    sh("docker", "exec", "-d", "e2e-client", "sh", "-c",
       "sstpc --log-level 4 --log-stderr --cert-warn --user alice --password %s %s:8448 noauth "
       "refuse-pap refuse-chap refuse-mschap refuse-eap nodefaultroute nodetach > /tmp/sstp.log 2>&1" % (pw, SERVER_A))
    if not wait_for("e2e-client", "ip -4 addr show | grep -q 'inet .* peer'", "the SSTP link", 25):
        return False
    dx("e2e-client", "dev=$(ip -o -4 addr show | awk '/ peer /{print $2}' | head -1); ip route add 198.18.0.0/24 dev $dev")
    return True


def t_ikev2(sub, pw, _):
    sh("docker", "cp", "e2e-server:/data/tls.crt", os.path.join(BUILD, "server.crt"))
    conf = """connections {
  omnitun {
    version = 2
    remote_addrs = %(srv)s
    vips = 0.0.0.0
    local { auth = eap-mschapv2
            id = alice
            eap_id = alice }
    remote { auth = pubkey
             id = %(srv)s
             certs = server.crt }
    children { net {
      remote_ts = 198.18.0.0/24
      esp_proposals = aes256-sha256-modp2048,aes128-sha1-modp1024,aes256gcm16-modp2048
      start_action = none } }
    proposals = aes256-sha256-modp2048,aes128-sha1-modp2048,aes256-sha384-modp3072,aes128-sha256-ecp256
    mobike = no
    fragmentation = no
  }
}
secrets { eap-alice { id = alice
                      secret = %(pw)s } }
""" % {"srv": SERVER_A, "pw": pw}
    path = os.path.join(BUILD, "swanctl.conf")
    open(path, "w", newline="\n").write(conf)
    dx("e2e-client", "mkdir -p /etc/swanctl/x509 /etc/swanctl/conf.d")
    sh("docker", "cp", os.path.join(BUILD, "server.crt"), "e2e-client:/etc/swanctl/x509/server.crt")
    sh("docker", "cp", path, "e2e-client:/etc/swanctl/swanctl.conf")
    sh("docker", "exec", "-d", "e2e-client", "sh", "-c", "/usr/lib/ipsec/charon > /tmp/charon.log 2>&1")
    if not wait_for("e2e-client", "swanctl --stats >/dev/null 2>&1", "strongSwan to start", 15):
        return False
    dx("e2e-client", "swanctl --load-all 2>&1")
    r = dx("e2e-client", "swanctl --initiate --child net --timeout 20 2>&1", check=False, timeout=60)
    print(r.stdout[-1500:])
    return r.returncode == 0


CLIENTS = {
    "openvpn-udp": t_openvpn("openvpn-udp"), "openvpn-tcp": t_openvpn("openvpn-tcp"),
    "wireguard": t_wireguard, "amneziawg": t_amneziawg, "sstp": t_sstp, "ikev2": t_ikev2,
}


def through_tunnel():
    r = dx("e2e-client", "curl -s --max-time 8 http://%s:8080/" % WEB_B, check=False)
    return r.stdout.strip()


def diagnostics(name):
    print("--- diagnostics for", name, "---")
    print(dx("e2e-client", "ip -br addr; ip route; tail -n 25 /tmp/*.log 2>/dev/null", check=False).stdout[-2500:])
    print(dx("e2e-server", "tail -n 25 /data/server.log; tail -n 15 /var/log/run.log", check=False).stdout[-2500:])


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("protocols", nargs="*", default=PROTOCOLS)
    ap.add_argument("--keep", action="store_true", help="leave the containers running")
    ap.add_argument("--no-build", action="store_true")
    args = ap.parse_args()
    if shutil.which("docker") is None or sh("docker", "info", check=False).returncode != 0:
        sys.exit("Docker is not running")
    if not args.no_build:
        build()
    start_world()
    sub, pw, _ = user_info()
    results = {}
    for name in args.protocols:
        print("\n=== %s ===" % name)
        start_client()
        ok = False
        try:
            ok = CLIENTS[name](sub, pw, None)
            answer = through_tunnel() if ok else ""
            ok = ok and answer == "client=%s" % SERVER_B
            print("  through the tunnel: %r (want client=%s)" % (answer, SERVER_B))
        except Exception as e:  # noqa: BLE001
            print("  error:", e)
        results[name] = ok
        if not ok:
            diagnostics(name)
        sh("docker", "rm", "-f", "e2e-client", check=False)
    # Accounting needs the stats file written (every 10 s).
    time.sleep(12)
    print("\n=== accounting (what the server counted) ===")
    u = dx("e2e-server", "omnitun-server --data /data user usage alice", check=False).stdout
    print(u)
    print(dx("e2e-server", "omnitun-server --data /data proto list", check=False).stdout)
    print("\n=== results ===")
    for k, v in results.items():
        print("  %-12s %s" % (k, "PASS" if v else "FAIL"))
    if not args.keep:
        cleanup()
    sys.exit(0 if all(results.values()) else 1)


if __name__ == "__main__":
    main()
