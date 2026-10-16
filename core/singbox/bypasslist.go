package singbox

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

// MaxBypassEntries bounds a bypass list so a huge file cannot slow every
// connection down or bloat the configuration.
const MaxBypassEntries = 50000

// BypassList is a set of destinations that go around the VPN (or, with
// Options.BypassOnly, the only ones that use it). Written one entry per line:
//
//	example.com            the domain and its subdomains
//	domain:example.com     the same
//	full:www.example.com   exactly this name
//	keyword:bank           any name containing this word
//	regexp:^ads?\..*       names matching the expression
//	203.0.113.7            an address
//	10.0.0.0/8             a range (IPv4 or IPv6)
//
// Blank lines and anything after # are ignored.
type BypassList struct {
	Suffix  []string
	Full    []string
	Keyword []string
	Regex   []string
	CIDRs   []netip.Prefix
}

// Empty reports whether the list has no entries.
func (b *BypassList) Empty() bool { return b == nil || b.Count() == 0 }

// Count is the number of entries.
func (b *BypassList) Count() int {
	if b == nil {
		return 0
	}
	return len(b.Suffix) + len(b.Full) + len(b.Keyword) + len(b.Regex) + len(b.CIDRs)
}

// BypassIssue is one line of a list that could not be used.
type BypassIssue struct {
	Line    int    `json:"line"` // 1-based
	Text    string `json:"text"`
	Message string `json:"message"`
}

func (i BypassIssue) Error() string {
	return fmt.Sprintf("line %d: %s (%s)", i.Line, i.Message, i.Text)
}

// ParseBypassList reads a list. Lines that cannot be used are reported and
// skipped; the rest of the list still counts.
func ParseBypassList(text string) (*BypassList, []BypassIssue) {
	var b BypassList
	var issues []BypassIssue
	seen := map[string]bool{}
	for n, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := raw
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		bad := func(msg string) { issues = append(issues, BypassIssue{Line: n + 1, Text: clip(raw), Message: msg}) }
		if b.Count() >= MaxBypassEntries {
			bad(fmt.Sprintf("more than %d entries: the rest is ignored", MaxBypassEntries))
			break
		}
		if seen[strings.ToLower(line)] {
			continue
		}
		seen[strings.ToLower(line)] = true

		kind, value, hasKind := strings.Cut(line, ":")
		switch {
		case hasKind && strings.EqualFold(kind, "regexp"):
			value = strings.TrimSpace(value)
			if value == "" {
				bad("empty expression")
			} else if _, err := regexp.Compile(value); err != nil {
				bad("not a valid expression")
			} else {
				b.Regex = append(b.Regex, value)
			}
		case hasKind && (strings.EqualFold(kind, "full") || strings.EqualFold(kind, "domain") || strings.EqualFold(kind, "keyword")):
			value = strings.ToLower(strings.TrimSpace(value))
			if !validName(value, strings.EqualFold(kind, "keyword")) {
				bad("not a valid name")
				continue
			}
			switch strings.ToLower(kind) {
			case "full":
				b.Full = append(b.Full, value)
			case "domain":
				b.Suffix = append(b.Suffix, value)
			default:
				b.Keyword = append(b.Keyword, value)
			}
		case hasKind && (strings.EqualFold(kind, "geosite") || strings.EqualFold(kind, "geoip")):
			bad("geo lists are not supported; list the domains or ranges")
		default:
			if p, err := netip.ParsePrefix(line); err == nil {
				b.CIDRs = append(b.CIDRs, p.Masked())
			} else if a, err := netip.ParseAddr(line); err == nil {
				b.CIDRs = append(b.CIDRs, netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()))
			} else if name := strings.ToLower(strings.TrimPrefix(line, ".")); validName(name, false) {
				b.Suffix = append(b.Suffix, name)
			} else {
				bad("not a domain, address or range")
			}
		}
	}
	return &b, issues
}

var nameRE = regexp.MustCompile(`^[a-z0-9_]([a-z0-9_-]*[a-z0-9_])?(\.[a-z0-9_]([a-z0-9_-]*[a-z0-9_])?)*$`)

// validName accepts host names (letters, digits, '-', '_', dots); a keyword
// may be any such word, also without a dot.
func validName(s string, keyword bool) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	if keyword {
		return nameRE.MatchString(strings.ReplaceAll(s, ".", "a"))
	}
	// A name whose last part is all digits is a mistyped address, not a domain.
	last := s[strings.LastIndexByte(s, '.')+1:]
	if strings.Trim(last, "0123456789") == "" {
		return false
	}
	return nameRE.MatchString(s)
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}

// rules are the sing-box rule bodies for the list: one per kind of entry,
// because the fields of one sing-box rule must all match.
func (b *BypassList) rules() []map[string]any {
	if b.Empty() {
		return nil
	}
	var out []map[string]any
	add := func(key string, v any, n int) {
		if n > 0 {
			out = append(out, map[string]any{key: v})
		}
	}
	add("domain_suffix", b.Suffix, len(b.Suffix))
	add("domain", b.Full, len(b.Full))
	add("domain_keyword", b.Keyword, len(b.Keyword))
	add("domain_regex", b.Regex, len(b.Regex))
	add("ip_cidr", prefixStrings(b.CIDRs), len(b.CIDRs))
	return out
}

// domainRules are the rules that match names only (what DNS rules can use).
func (b *BypassList) domainRules() []map[string]any {
	var out []map[string]any
	for _, r := range b.rules() {
		if _, ip := r["ip_cidr"]; !ip {
			out = append(out, r)
		}
	}
	return out
}
