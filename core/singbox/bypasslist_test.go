package singbox

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
)

func TestParseBypassListEveryKind(t *testing.T) {
	list, issues := ParseBypassList(strings.Join([]string{
		"# my banks and the office",
		"",
		"Example.COM",
		"domain:Other.example",
		".dotted.example",
		"full:www.exact.example   # only this name",
		"keyword:bank",
		"regexp:^ads?\\..*",
		"203.0.113.7",
		"10.1.2.3/8",
		"2001:db8::/32",
		"::ffff:198.51.100.9",
		"example.com", // a duplicate is not counted twice
	}, "\r\n"))
	if len(issues) != 0 {
		t.Fatalf("issues: %v", issues)
	}
	eq := func(name string, got, want []string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	eq("suffix", list.Suffix, []string{"example.com", "other.example", "dotted.example"})
	eq("full", list.Full, []string{"www.exact.example"})
	eq("keyword", list.Keyword, []string{"bank"})
	eq("regex", list.Regex, []string{`^ads?\..*`})
	eq("cidrs", prefixStrings(list.CIDRs), []string{"203.0.113.7/32", "10.0.0.0/8", "2001:db8::/32", "198.51.100.9/32"})
	if list.Count() != 10 {
		t.Errorf("count = %d", list.Count())
	}
}

func TestParseBypassListReportsBadLinesAndKeepsTheRest(t *testing.T) {
	list, issues := ParseBypassList(strings.Join([]string{
		"good.example",
		"not a name at all",
		"regexp:(unclosed",
		"geosite:category-ads",
		"full:bad name",
		"1.2.3.999",
		"fine.example # trailing comment",
	}, "\n"))
	if list.Count() != 2 {
		t.Fatalf("kept %d entries: %+v", list.Count(), list)
	}
	wantLines := []int{2, 3, 4, 5, 6}
	if len(issues) != len(wantLines) {
		t.Fatalf("issues: %v", issues)
	}
	for i, n := range wantLines {
		if issues[i].Line != n || issues[i].Message == "" || issues[i].Text == "" {
			t.Errorf("issue %d: %+v", i, issues[i])
		}
	}
	if !strings.Contains(issues[2].Message, "geo") {
		t.Errorf("geo message: %q", issues[2].Message)
	}
	if _, issues := ParseBypassList(""); len(issues) != 0 {
		t.Error("an empty list has issues")
	}
}

func TestParseBypassListIsBounded(t *testing.T) {
	var b strings.Builder
	for i := range MaxBypassEntries + 10 {
		b.WriteString("h")
		b.WriteString(strings.Repeat("a", i%7))
		b.WriteString(strconv.Itoa(i))
		b.WriteString(".example\n")
	}
	list, issues := ParseBypassList(b.String())
	if list.Count() != MaxBypassEntries || len(issues) != 1 || !strings.Contains(issues[0].Message, "more than") {
		t.Fatalf("count %d, issues %v", list.Count(), issues)
	}
}

// One rule per kind of entry (the fields of a sing-box rule all have to match),
// direct for a bypass list, the VPN for "only these".
func TestBypassListRoutesAndDNS(t *testing.T) {
	p := testProfile(t)
	list, _ := ParseBypassList("bank.example\nfull:a.b.example\n203.0.113.0/24\n")

	cfg, _, err := Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list})
	if err != nil {
		t.Fatal(err)
	}
	route := cfg["route"].(obj)
	if route["final"] != TagSelect {
		t.Fatalf("final = %v", route["final"])
	}
	rules := route["rules"].([]any)
	var suffix, full, cidr obj
	for _, r := range rules {
		o := r.(obj)
		switch {
		case o["domain_suffix"] != nil:
			suffix = o
		case o["domain"] != nil:
			full = o
		case o["ip_cidr"] != nil && o["outbound"] == TagDirect && len(o) == 2 && contains(o["ip_cidr"].([]string), "203.0.113.0/24"):
			cidr = o
		}
	}
	for name, r := range map[string]obj{"suffix": suffix, "full": full, "cidr": cidr} {
		if r == nil || r["outbound"] != TagDirect || len(r) != 2 {
			t.Errorf("%s rule: %v", name, r)
		}
	}
	// The listed names are resolved locally, like they are reached.
	dns := cfg["dns"].(obj)
	dnsRules := dns["rules"].([]any)
	if len(dnsRules) != 2 || dnsRules[0].(obj)["server"] != "local" || dns["final"] != "remote" {
		t.Fatalf("dns: %v", dns)
	}
	// The address range also stays out of the VPN's routes.
	tun := cfg["inbounds"].([]any)[0].(obj)
	if !contains(tun["route_exclude_address"].([]string), "203.0.113.0/24") {
		t.Errorf("route_exclude_address = %v", tun["route_exclude_address"])
	}

	// "Only these": they use the VPN, the rest goes direct and is resolved locally.
	cfg, _, err = Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list, BypassOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	route = cfg["route"].(obj)
	if route["final"] != TagDirect {
		t.Fatalf("final = %v", route["final"])
	}
	for _, r := range route["rules"].([]any) {
		if o := r.(obj); o["domain_suffix"] != nil && o["outbound"] != TagSelect {
			t.Errorf("a listed name does not use the VPN: %v", o)
		}
	}
	dns = cfg["dns"].(obj)
	if dns["final"] != "local" || dns["rules"].([]any)[0].(obj)["server"] != "remote" {
		t.Fatalf("dns: %v", dns)
	}
	if _, ok := cfg["inbounds"].([]any)[0].(obj)["route_exclude_address"].([]string); ok {
		if contains(cfg["inbounds"].([]any)[0].(obj)["route_exclude_address"].([]string), "203.0.113.0/24") {
			t.Error("'only these' excludes the listed range from the VPN")
		}
	}

	// No list: nothing changes.
	cfg, _, _ = Build(p.AsSlice(), Options{Mode: ModeProxy})
	if cfg["route"].(obj)["final"] != TagSelect || cfg["dns"].(obj)["rules"] != nil {
		t.Error("an empty bypass list changed the configuration")
	}
}

func testProfile(t *testing.T) *profile.Profile {
	t.Helper()
	p := &profile.Profile{Protocol: profile.Trojan, Name: "t", Server: "tr.example.com", Port: 443, Password: "pw"}
	p.EnsureID()
	return p
}

func contains(list []string, v string) bool {
	return slices.Contains(list, v)
}
