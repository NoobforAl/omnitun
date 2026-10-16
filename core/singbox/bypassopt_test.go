package singbox

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

func TestOptimizedDropsCoveredEntries(t *testing.T) {
	list, issues := ParseBypassList(`
example.com
a.example.com
b.a.example.com
other.org
full:www.example.com
full:solo.net
keyword:bank
mybank.com
keyword:mybank
10.0.0.0/8
10.1.0.0/16
192.168.0.0/25
192.168.0.128/25
`)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	o := list.Optimized()
	if want := []string{"example.com", "other.org"}; !slices.Equal(o.Suffix, want) {
		t.Errorf("suffix = %v, want %v", o.Suffix, want)
	}
	if want := []string{"solo.net"}; !slices.Equal(o.Full, want) {
		t.Errorf("full = %v, want %v", o.Full, want)
	}
	if want := []string{"bank"}; !slices.Equal(o.Keyword, want) {
		t.Errorf("keyword = %v, want %v", o.Keyword, want)
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("192.168.0.0/24")}
	if !slices.Equal(o.CIDRs, want) {
		t.Errorf("cidrs = %v, want %v", o.CIDRs, want)
	}
	if (*BypassList)(nil).Optimized() != nil {
		t.Error("nil list")
	}
}

func TestFitExclusionsRespectsBudget(t *testing.T) {
	local := BypassPrefixes(nil)
	var many []netip.Prefix
	for i := range 5000 {
		many = append(many, netip.MustParsePrefix(fmt.Sprintf("%d.%d.%d.0/24", 11+i/(256*4), (i/4)%256, (i%4)*50)))
	}
	big := netip.MustParsePrefix("8.8.0.0/16")
	fit := FitExclusions(local, append(many, big), RouteBudget)
	if !slices.Contains(fit, big) {
		t.Error("the biggest range was dropped first")
	}
	if len(fit) >= len(many) {
		t.Fatalf("nothing was dropped: %d", len(fit))
	}
	if routes := len(SubtractPrefixes([]netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}, fit)); routes > RouteBudget {
		t.Errorf("%d routes, budget %d", routes, RouteBudget)
	}
	// A short list is kept whole.
	few := []netip.Prefix{netip.MustParsePrefix("203.0.113.0/24")}
	if got := FitExclusions(local, few, RouteBudget); !slices.Contains(got, few[0]) {
		t.Errorf("short list lost: %v", got)
	}
}

func TestBypassOnlyRangesStayOutOfTheTunnel(t *testing.T) {
	p := testProfile(t)
	list, _ := ParseBypassList("203.0.113.0/24\n198.51.100.7\n")
	cfg, _, err := Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list, BypassOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	tun := cfg["inbounds"].([]any)[0].(obj)
	routes, _ := tun["route_address"].([]string)
	for _, want := range []string{"203.0.113.0/24", "198.51.100.7/32", "172.19.0.0/30"} {
		if !slices.Contains(routes, want) {
			t.Errorf("route_address %v lacks %s", routes, want)
		}
	}
	// With a name in the list everything has to be inspected, so everything enters.
	list, _ = ParseBypassList("203.0.113.0/24\nexample.com\n")
	cfg, _, _ = Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list, BypassOnly: true})
	if _, ok := cfg["inbounds"].([]any)[0].(obj)["route_address"]; ok {
		t.Error("route_address set although names are listed")
	}
}

func TestRulesAreCheapestFirstAndSniffOnlyForNames(t *testing.T) {
	p := testProfile(t)
	has := func(cfg obj, action string) bool {
		for _, r := range cfg["route"].(obj)["rules"].([]any) {
			if r.(obj)["action"] == action {
				return true
			}
		}
		return false
	}
	// Addresses only: no packet is ever inspected.
	list, _ := ParseBypassList("203.0.113.0/24\n")
	cfg, _, _ := Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list})
	if has(cfg, "sniff") {
		t.Error("sniffing without a listed name")
	}
	if first := cfg["route"].(obj)["rules"].([]any)[0].(obj); first["action"] != "hijack-dns" {
		t.Errorf("first rule = %v", first)
	}
	// With names: addresses are decided before the sniff, names after it.
	list, _ = ParseBypassList("203.0.113.0/24\nexample.com\nkeyword:bank\nregexp:^ads?[.]\n")
	cfg, _, _ = Build(p.AsSlice(), Options{Mode: ModeTUN, Bypass: list})
	var order []string
	for _, r := range cfg["route"].(obj)["rules"].([]any) {
		o := r.(obj)
		switch {
		case o["action"] == "sniff":
			order = append(order, "sniff")
		case o["ip_cidr"] != nil && o["outbound"] == TagDirect:
			order = append(order, "cidr")
		case o["domain_suffix"] != nil:
			order = append(order, "suffix")
		case o["domain_keyword"] != nil:
			order = append(order, "keyword")
		case o["domain_regex"] != nil:
			order = append(order, "regex")
		}
	}
	if got := strings.Join(order, ","); got != "cidr,cidr,sniff,suffix,keyword,regex" {
		t.Errorf("order = %s", got)
	}
}
