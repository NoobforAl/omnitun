package api

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Serve runs the API on host:port until ctx is done. With a non-nil tlsConfig
// it speaks HTTPS.
func Serve(ctx context.Context, host string, port int, h http.Handler, tlsConfig *tls.Config, logf func(string, ...any)) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	scheme := "http"
	if tlsConfig != nil {
		ln, scheme = tls.NewListener(ln, tlsConfig), "https"
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	logf("web API on %s://%s (docs at /docs)", scheme, ln.Addr())
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) && tlsConfig == nil {
		logf("warning: the web API listens on a public address over plain HTTP; turn on its TLS (omnitun-server api tls on), or use an SSH tunnel")
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
