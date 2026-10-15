package engine

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// Every protocol the importer understands must produce a config sing-box accepts.
func TestGeneratedConfigsPassSchema(t *testing.T) {
	for _, link := range testLinks() {
		p, err := importer.ParseLink(link)
		if err != nil {
			t.Fatalf("%s: %v", link, err)
		}
		for _, mode := range []singbox.Mode{singbox.ModeProxy, singbox.ModeTUN} {
			cfg, skipped, err := singbox.Build(p.AsSlice(), singbox.Options{Mode: mode})
			if err != nil || len(skipped) > 0 {
				t.Fatalf("%s (%s): build err=%v skipped=%v", p.Name, mode, err, skipped)
			}
			if err := CheckConfig(cfg); err != nil {
				t.Errorf("%s (%s): sing-box rejected config: %v", p.Name, mode, err)
			}
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// testLinks covers every protocol the sing-box engine runs.
func testLinks() []string {
	ss := base64.RawURLEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pass"))
	vmess := base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"vm","add":"vm.example.com","port":"443","id":"b831381d-6324-4d53-ad4f-8cda48b30811","aid":"0","net":"ws","path":"/v?ed=2048","host":"vm.example.com","tls":"tls"}`))
	return []string{
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?security=reality&sni=www.microsoft.com&fp=chrome&pbk=jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0&sid=12ab&flow=xtls-rprx-vision#reality",
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?security=tls&type=grpc&serviceName=gun&sni=example.com#grpc",
		"vmess://" + vmess,
		"trojan://secret@tr.example.com:443?type=httpupgrade&path=%2Fup&host=tr.example.com#tr",
		"ss://" + ss + "@ss.example.com:8388#ss",
		"hy2://pw@hy.example.com:443?sni=hy.example.com&obfs=salamander&obfs-password=ob#hy",
		"tuic://b831381d-6324-4d53-ad4f-8cda48b30811:pw@tu.example.com:443?congestion_control=bbr&sni=tu.example.com#tu",
		"wireguard://yAnz5TF%2BlXXJte14tji3zlMNq%2Bhd2rYUIgJBgB3fBmk%3D@wg.example.com:51820?publickey=xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg%3D&address=10.0.0.2%2F32&reserved=1,2,3#wg",
	}
}

func TestDNSVariantsPassSchema(t *testing.T) {
	p, err := importer.ParseLink("trojan://pw@example.com:443#t")
	if err != nil {
		t.Fatal(err)
	}
	for _, dns := range []string{
		"", "https://1.1.1.1/dns-query", "https://dns.example.com/custom", "tls://1.1.1.1",
		"tcp://1.1.1.1", "udp://8.8.8.8:53", "9.9.9.9",
	} {
		cfg, _, err := singbox.Build(p.AsSlice(), singbox.Options{DNS: dns})
		if err != nil {
			t.Fatalf("%q: %v", dns, err)
		}
		if err := CheckConfig(cfg); err != nil {
			t.Errorf("%q: sing-box rejected config: %v", dns, err)
		}
	}
	if _, _, err := singbox.Build(p.AsSlice(), singbox.Options{DNS: "ftp://x"}); err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestSplitTunnelPassesSchema(t *testing.T) {
	p, err := importer.ParseLink("trojan://pw@example.com:443#t")
	if err != nil {
		t.Fatal(err)
	}
	for _, opts := range []singbox.Options{
		{Mode: singbox.ModeTUN, IncludePackages: []string{"org.telegram.messenger"}},
		{Mode: singbox.ModeTUN, ExcludePackages: []string{"com.android.chrome", "ir.mci.ecareapp"}},
	} {
		cfg, _, err := singbox.Build(p.AsSlice(), opts)
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckConfig(cfg); err != nil {
			t.Fatalf("sing-box rejected split-tunnel config: %v", err)
		}
		tun := cfg["inbounds"].([]any)[0].(map[string]any)
		if _, inc := tun["include_package"]; inc == (len(opts.IncludePackages) == 0) {
			t.Fatalf("include/exclude not applied: %+v", tun)
		}
	}
}
