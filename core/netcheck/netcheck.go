// Package netcheck decides whether the device has a working network at all,
// so the UI can say "no internet" instead of blaming (and failing over) a profile.
package netcheck

import (
	"context"
	"errors"
	"net"
	"time"
)

// ErrOffline means no probe succeeded.
var ErrOffline = errors.New("no internet connection")

// probes mixes anycast resolvers and a DNS lookup through the system resolver,
// so a network where some public IPs are blocked still counts as online.
var probes = []func(ctx context.Context) error{
	dialProbe("1.1.1.1:443"),
	dialProbe("8.8.8.8:443"),
	dialProbe("9.9.9.9:443"),
	dialProbe("208.67.222.222:443"),
	lookupProbe("www.microsoft.com"),
	lookupProbe("www.apple.com"),
}

func dialProbe(addr string) func(context.Context) error {
	return func(ctx context.Context) error {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			conn.Close()
		}
		return err
	}
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

// Online returns nil as soon as any probe succeeds, or ErrOffline when all fail
// or ctx expires. Probes run in parallel with a 4s cap.
func Online(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	results := make(chan error, len(probes))
	for _, p := range probes {
		go func() { results <- p(ctx) }()
	}
	for range probes {
		if <-results == nil {
			return nil
		}
	}
	return ErrOffline
}
