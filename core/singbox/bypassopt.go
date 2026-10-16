package singbox

import (
	"slices"
	"sort"
	"strings"

	"go4.org/netipx"

	"net/netip"
)

// RouteBudget bounds how many routes the VPN interface is given once the
// excluded ranges are taken out of the default routes. Every route is state in
// the kernel and in the app; a few hundred cost nothing, tens of thousands
// slow every connection setup down.
const RouteBudget = 512

// Optimized returns an equivalent list without entries that another entry
// already covers, so the matchers stay small and fast:
//
//   - a name under another listed domain (a.example.com next to example.com),
//   - a name that contains a listed keyword (it matches the keyword anyway),
//   - a keyword that contains a shorter listed keyword,
//   - ranges inside other ranges, with neighbouring ranges merged.
//
// The matching result is the same for every destination.
func (b *BypassList) Optimized() *BypassList {
	if b == nil {
		return nil
	}
	out := &BypassList{Regex: dedupe(b.Regex)}

	// Keywords: drop a keyword that contains a shorter one.
	kws := dedupe(b.Keyword)
	sort.Slice(kws, func(i, j int) bool { return len(kws[i]) < len(kws[j]) })
	for _, k := range kws {
		covered := false
		for _, kept := range out.Keyword {
			if strings.Contains(k, kept) {
				covered = true
				break
			}
		}
		if !covered {
			out.Keyword = append(out.Keyword, k)
		}
	}
	hasKeyword := func(name string) bool {
		for _, k := range out.Keyword {
			if strings.Contains(name, k) {
				return true
			}
		}
		return false
	}

	// Suffixes: shortest first, so a parent is seen before its children.
	suffix := map[string]bool{}
	underSuffix := func(name string) bool {
		for rest := name; ; {
			if suffix[rest] {
				return true
			}
			_, after, ok := strings.Cut(rest, ".")
			if !ok {
				return false
			}
			rest = after
		}
	}
	names := dedupe(b.Suffix)
	sort.Slice(names, func(i, j int) bool { return strings.Count(names[i], ".") < strings.Count(names[j], ".") })
	for _, n := range names {
		// The name itself may already be listed through a parent; check parents only.
		_, parent, hasParent := strings.Cut(n, ".")
		if hasKeyword(n) || (hasParent && underSuffix(parent)) {
			continue
		}
		suffix[n] = true
		out.Suffix = append(out.Suffix, n)
	}
	slices.Sort(out.Suffix)

	// Exact names: a suffix or keyword may already match them.
	for _, n := range dedupe(b.Full) {
		if !hasKeyword(n) && !underSuffix(n) {
			out.Full = append(out.Full, n)
		}
	}
	slices.Sort(out.Full)

	out.CIDRs = MergePrefixes(b.CIDRs)
	return out
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// MergePrefixes returns the smallest set of ranges that covers the same
// addresses: duplicates and nested ranges disappear, neighbours join.
func MergePrefixes(in []netip.Prefix) []netip.Prefix {
	if len(in) < 2 {
		return in
	}
	var b netipx.IPSetBuilder
	for _, p := range in {
		b.AddPrefix(p.Masked())
	}
	set, err := b.IPSet()
	if err != nil {
		return in
	}
	return set.Prefixes()
}

// FitExclusions returns base plus as many of the listed ranges as fit in the
// route budget, biggest ranges first (they keep the most traffic out of the
// tunnel for the fewest routes). Ranges that do not fit are still caught by
// the routing rules, just one step later, inside the tunnel.
func FitExclusions(base, listed []netip.Prefix, budget int) []netip.Prefix {
	listed = MergePrefixes(listed)
	sort.SliceStable(listed, func(i, j int) bool { return listed[i].Bits() < listed[j].Bits() })

	// Routes needed to reach everything except base and the first n listed ranges.
	cost := func(n int) int {
		var b netipx.IPSetBuilder
		b.AddPrefix(netip.MustParsePrefix("0.0.0.0/0"))
		b.AddPrefix(netip.MustParsePrefix("::/0"))
		for _, p := range base {
			b.RemovePrefix(p.Masked())
		}
		for _, p := range listed[:n] {
			b.RemovePrefix(p)
		}
		set, err := b.IPSet()
		if err != nil {
			return budget + 1
		}
		return len(set.Prefixes())
	}
	// Largest n that still fits (the cost only grows with n, up to small dips
	// when neighbours join, which a binary search tolerates).
	lo, hi := 0, len(listed)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if cost(mid) <= budget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return append(slices.Clone(base), listed[:lo]...)
}
