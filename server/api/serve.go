package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Serve runs the API on host:port until ctx is done.
func Serve(ctx context.Context, host string, port int, h http.Handler, logf func(string, ...any)) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	logf("web API on http://%s (docs at /docs)", ln.Addr())
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		logf("warning: the web API listens on a public address over plain HTTP; put it behind a TLS proxy or use an SSH tunnel")
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
