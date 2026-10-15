// Package netcheck decides whether the device has a working network at all,
// so the UI can say "no internet" instead of blaming (and failing over) a profile.
package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// ErrOffline means no probe succeeded.
var ErrOffline = errors.New("no internet connection")

// DialFunc dials a TCP address.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// dialTargets mixes several anycast resolvers so a network where some public
// IPs are blocked still counts as online.
var dialTargets = []string{
	"1.1.1.1:443",
	"8.8.8.8:443",
	"9.9.9.9:443",
	"208.67.222.222:443",
	"1.1.1.1:853",
}

var lookupHosts = []string{"www.microsoft.com", "www.apple.com"}

// Online checks with the system dialer and resolver. Use it before a tunnel
// exists; once one is up, use OnlineDial with a dialer that bypasses it.
func Online(ctx context.Context) error {
	var d net.Dialer
	probes := dialProbes(d.DialContext)
	for _, h := range lookupHosts {
		probes = append(probes, lookupProbe(h))
	}
	return run(ctx, probes)
}

// OnlineDial checks by dialing through dial only (no DNS), for use while a
// tunnel is running.
func OnlineDial(ctx context.Context, dial DialFunc) error {
	return run(ctx, dialProbes(dial))
}

func dialProbes(dial DialFunc) []func(context.Context) error {
	probes := make([]func(context.Context) error, 0, len(dialTargets))
	for _, addr := range dialTargets {
		probes = append(probes, func(ctx context.Context) error {
			conn, err := dial(ctx, "tcp", addr)
			if err == nil {
				conn.Close()
			}
			return err
		})
	}
	return probes
}

func lookupProbe(host string) func(context.Context) error {
	return func(ctx context.Context) error {
		addrs, err := net.DefaultResolver.LookupHost(ctx, host)
		if err == nil && len(addrs) == 0 {
			err = errors.New("empty answer")
		}
		return err
	}
}

// run returns nil as soon as any probe succeeds, or ErrOffline when all fail
// or ctx expires. Probes run in parallel with a 4s cap.
func run(ctx context.Context, probes []func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	results := make(chan error, len(probes))
	for _, p := range probes {
		go func() { results <- p(ctx) }()
	}
	var first error
	for range probes {
		err := <-results
		if err == nil {
			return nil
		}
		if first == nil {
			first = err
		}
	}
	return fmt.Errorf("%w (%v)", ErrOffline, first)
}
