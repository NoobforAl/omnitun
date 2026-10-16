package install

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type fakeRunner struct {
	out     map[string]string // command line -> output
	failing map[string]bool
	ran     []string
	active  map[string]bool
}

func (f *fakeRunner) Run(argv ...string) (string, error) {
	line := strings.Join(argv, " ")
	f.ran = append(f.ran, line)
	if f.failing[line] {
		return "boom", errors.New("exit status 1")
	}
	return f.out[line], nil
}

func (f *fakeRunner) Active(argv ...string) bool { return f.active[strings.Join(argv, " ")] }

var ports = []Port{
	{443, "tcp", "vless-reality"}, {8388, "tcp", "shadowsocks"}, {8388, "udp", "shadowsocks"}, {51820, "udp", "wireguard"},
}

func commands(p Plan) []string {
	var out []string
	for _, s := range p.Steps {
		if s.Command != nil {
			out = append(out, strings.Join(s.Command, " "))
		}
	}
	return out
}

func contains(list []string, s string) bool {
	return slices.Contains(list, s)
}

func TestDetectFirewall(t *testing.T) {
	if got := Detect(&fakeRunner{out: map[string]string{"ufw status": "Status: active\n..."}}); got != "ufw" {
		t.Errorf("active ufw: %s", got)
	}
	if got := Detect(&fakeRunner{out: map[string]string{"ufw status": "Status: inactive"}, active: map[string]bool{"firewall-cmd --state": true}}); got != "firewalld" {
		t.Errorf("firewalld: %s", got)
	}
	if got := Detect(&fakeRunner{out: map[string]string{"ufw status": "Status: inactive"}}); got != "none" {
		t.Errorf("nothing active: %s", got)
	}
}

func TestPlanWithUFW(t *testing.T) {
	plan, err := BuildPlan(Options{Binary: "/tmp/omnitun-server", DataDir: "/var/lib/omnitun-server", User: "omnitun", Ports: ports}, "ufw")
	if err != nil {
		t.Fatal(err)
	}
	cmds := commands(plan)
	for _, want := range []string{
		"useradd --system --home-dir /var/lib/omnitun-server --shell /usr/sbin/nologin omnitun",
		"mkdir -p /var/lib/omnitun-server",
		"chown -R omnitun:omnitun /var/lib/omnitun-server",
		"chmod 700 /var/lib/omnitun-server",
		"install -m 0755 /tmp/omnitun-server /usr/local/bin/omnitun-server",
		"systemctl daemon-reload",
		"ufw allow 443/tcp comment omnitun: vless-reality",
		"ufw allow 8388/udp comment omnitun: shadowsocks",
		"ufw allow 51820/udp comment omnitun: wireguard",
		"systemctl enable --now omnitun-server",
	} {
		if !contains(cmds, want) {
			t.Errorf("missing step %q in:\n%s", want, strings.Join(cmds, "\n"))
		}
	}
	// The unit is written before the service is started, and the firewall is
	// opened before it too.
	order := func(prefix string) int {
		for i, c := range cmds {
			if strings.HasPrefix(c, prefix) {
				return i
			}
		}
		return -1
	}
	if order("systemctl daemon-reload") > order("systemctl enable") || order("ufw allow 443") > order("systemctl enable") {
		t.Errorf("wrong order:\n%s", strings.Join(cmds, "\n"))
	}
	var unit *Step
	for i := range plan.Steps {
		if plan.Steps[i].File == UnitPath {
			unit = &plan.Steps[i]
		}
	}
	if unit == nil {
		t.Fatal("no unit file step")
	}
	for _, want := range []string{
		"ExecStart=/usr/local/bin/omnitun-server --data /var/lib/omnitun-server run",
		"User=omnitun", "AmbientCapabilities=CAP_NET_BIND_SERVICE", "NoNewPrivileges=true",
		"ReadWritePaths=/var/lib/omnitun-server", "Restart=on-failure", "WantedBy=multi-user.target",
	} {
		if !strings.Contains(unit.Content, want) {
			t.Errorf("unit lacks %q:\n%s", want, unit.Content)
		}
	}
}

func TestPlanWithFirewalldAndRoot(t *testing.T) {
	plan, err := BuildPlan(Options{Binary: "/opt/ots", DataDir: "/srv/ots", Ports: ports[:2]}, "firewalld")
	if err != nil {
		t.Fatal(err)
	}
	cmds := commands(plan)
	for _, want := range []string{"firewall-cmd --permanent --add-port=443/tcp", "firewall-cmd --permanent --add-port=8388/tcp", "firewall-cmd --reload"} {
		if !contains(cmds, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, c := range cmds {
		if strings.HasPrefix(c, "useradd") || strings.HasPrefix(c, "chown") {
			t.Errorf("a root service must not create or chown a user: %s", c)
		}
	}
}

func TestPlanWithoutAFirewallExplainsWhatToAllow(t *testing.T) {
	plan, _ := BuildPlan(Options{Binary: "/opt/ots", DataDir: "/srv/ots", Ports: ports}, "none")
	for _, c := range commands(plan) {
		if strings.HasPrefix(c, "ufw") || strings.HasPrefix(c, "firewall-cmd") {
			t.Fatalf("a firewall command without a firewall: %s", c)
		}
	}
	notes := strings.Join(plan.Notes, "\n")
	for _, want := range []string{"TCP: 443, 8388", "UDP: 8388, 51820", "security group"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	// An explicit choice wins over detection.
	forced, _ := BuildPlan(Options{Binary: "/opt/ots", DataDir: "/srv/ots", Ports: ports, Firewall: "none"}, "ufw")
	if contains(commands(forced), "ufw allow 443/tcp comment omnitun: vless-reality") {
		t.Error("--firewall none still used ufw")
	}
}

func TestPlanRejectsUnsafeInput(t *testing.T) {
	for name, o := range map[string]Options{
		"no binary":          {DataDir: "/srv/x"},
		"relative data dir":  {Binary: "/b", DataDir: "srv/x"},
		"space in data dir":  {Binary: "/b", DataDir: "/srv/my data"},
		"shell in data dir":  {Binary: "/b", DataDir: "/srv/x;rm -rf /"},
		"quote in user":      {Binary: "/b", DataDir: "/srv/x", User: "a\"b"},
		"newline in bin dir": {Binary: "/b", DataDir: "/srv/x", BinDir: "/usr/local/bin\n[Service]"},
	} {
		if _, err := BuildPlan(o, "none"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestRemovalPlan(t *testing.T) {
	cmds := commands(BuildRemovalPlan(Options{DataDir: "/srv/ots", Ports: ports[:1]}, "ufw"))
	for _, want := range []string{"systemctl disable --now omnitun-server", "rm -f " + UnitPath, "ufw delete allow 443/tcp"} {
		if !contains(cmds, want) {
			t.Errorf("missing %q in %v", want, cmds)
		}
	}
	for _, c := range cmds {
		if strings.Contains(c, "/srv/ots") || strings.HasPrefix(c, "rm -rf") {
			t.Errorf("removal touches the data: %s", c)
		}
	}
}

func TestExecuteStopsOnFailureUnlessIgnored(t *testing.T) {
	plan := Plan{Steps: []Step{
		{Description: "first", Command: []string{"a"}},
		{Description: "may fail", Command: []string{"b"}, IgnoreError: true},
		{Description: "must work", Command: []string{"c"}},
		{Description: "never reached", Command: []string{"d"}},
	}}
	r := &fakeRunner{failing: map[string]bool{"b": true, "c": true}}
	var out bytes.Buffer
	err := Execute(plan, r, &out)
	if err == nil || !strings.Contains(err.Error(), "must work") || strings.Join(r.ran, ",") != "a,b,c" {
		t.Fatalf("err = %v ran = %v", err, r.ran)
	}
	if !strings.Contains(out.String(), "[3/4] must work") {
		t.Errorf("progress output: %s", out.String())
	}
}

func TestExecuteWritesFilesAndDescribeShowsThem(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "etc", "unit.service")
	plan := Plan{Steps: []Step{{Description: "write the unit", File: target, FileMode: 0o644, Content: "[Unit]\nDescription=x\n"}}, Notes: []string{"hello"}}
	if err := Execute(plan, &fakeRunner{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "[Unit]\nDescription=x\n" {
		t.Fatalf("file: %q %v", data, err)
	}
	var b bytes.Buffer
	Describe(plan, &b)
	if !strings.Contains(b.String(), "#   write "+target) || !strings.Contains(b.String(), "#   | Description=x") || !strings.Contains(b.String(), "note: hello") {
		t.Errorf("describe:\n%s", b.String())
	}
}

func TestHostServersNeedCapabilitiesAndForwarding(t *testing.T) {
	plan, err := BuildPlan(Options{Binary: "/opt/ots", DataDir: "/srv/ots", User: "omnitun", HostServers: true}, "none")
	if err != nil {
		t.Fatal(err)
	}
	var unit, sysctl *Step
	for i := range plan.Steps {
		switch plan.Steps[i].File {
		case UnitPath:
			unit = &plan.Steps[i]
		case SysctlPath:
			sysctl = &plan.Steps[i]
		}
	}
	if unit == nil || !strings.Contains(unit.Content, "AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_NET_ADMIN") ||
		!strings.Contains(unit.Content, "CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_NET_ADMIN") {
		t.Fatalf("unit lacks CAP_NET_ADMIN:\n%v", unit)
	}
	if sysctl == nil || sysctl.Content != "net.ipv4.ip_forward=1\n" || !contains(commands(plan), "sysctl --system") {
		t.Fatalf("forwarding not enabled: %+v / %v", sysctl, commands(plan))
	}

	// Without SSTP/IKEv2 the service keeps the smaller capability set and
	// does not touch the host's forwarding.
	plain, _ := BuildPlan(Options{Binary: "/opt/ots", DataDir: "/srv/ots", User: "omnitun"}, "none")
	for _, s := range plain.Steps {
		if s.File == SysctlPath || strings.Contains(s.Content, "CAP_NET_ADMIN") {
			t.Fatalf("host-server setup without host servers: %+v", s)
		}
	}
	if contains(commands(plain), "sysctl --system") {
		t.Fatal("sysctl applied without host servers")
	}
}
