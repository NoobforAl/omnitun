// Package tor runs Tor for Omnitun: the Arti client (the Rust `omnitun-tor`
// binary from tor/) behind a local SOCKS5 port, plus bridge pluggable
// transports run in-process in Go (obfs4, WebTunnel, Snowflake, meek).
package tor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
)

// Config starts Tor.
type Config struct {
	// Bin is the omnitun-tor executable; empty uses FindBinary.
	Bin string
	// DataDir keeps Arti's state and directory cache between runs (faster
	// bootstraps).
	DataDir string
	// Bridges are bridge lines; empty connects to Tor directly.
	Bridges []string
	// Dial makes the pluggable transports' outbound connections.
	Dial DialFunc
	// OnBootstrap receives bootstrap progress (0-100) and Arti's description.
	OnBootstrap func(percent int, status string)
	// Logf receives Arti's warnings.
	Logf func(format string, args ...any)
}

// Process is a running Tor client.
type Process struct {
	// SocksAddr is the local SOCKS5 address (127.0.0.1:port).
	SocksAddr string

	cmd   *exec.Cmd
	stdin io.WriteCloser
	pts   *Transports
	ready chan error
	done  chan struct{}
	once  sync.Once
	err   error
}

type event struct {
	Event   string `json:"event"`
	Socks   string `json:"socks"`
	Percent int    `json:"percent"`
	Status  string `json:"status"`
	Blocked string `json:"blocked"`
	Message string `json:"message"`
}

// FindBinary locates omnitun-tor: $OMNITUN_TOR_BIN, then next to the running
// executable, then $PATH.
func FindBinary() (string, error) {
	if p := os.Getenv("OMNITUN_TOR_BIN"); p != "" {
		return p, nil
	}
	name := "omnitun-tor"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", errors.New("the Tor client (omnitun-tor) is not installed")
}

// Start launches the transports and Arti; it returns once Arti's SOCKS port
// is listening. Use Ready to wait for the bootstrap.
func Start(cfg Config) (*Process, error) {
	bin := cfg.Bin
	if bin == "" {
		var err error
		if bin, err = FindBinary(); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}
	args := []string{"--data", cfg.DataDir, "--socks-port", "0"}
	var bridges []string
	for _, b := range cfg.Bridges {
		if b = NormalizeBridge(b); b != "" {
			bridges = append(bridges, b)
			args = append(args, "--bridge", b)
		}
	}
	p := &Process{ready: make(chan error, 1), done: make(chan struct{})}
	if names := TransportsFor(bridges); len(names) > 0 {
		pts, err := StartTransports(filepath.Join(cfg.DataDir, "pt"), names, cfg.Dial)
		if err != nil {
			return nil, err
		}
		p.pts = pts
		keys := make([]string, 0, len(pts.Addrs))
		for k := range pts.Addrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			args = append(args, "--transport", k+"="+pts.Addrs[k])
		}
	}

	p.cmd = exec.Command(bin, args...)
	hideWindow(p.cmd)
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		p.closeTransports()
		return nil, err
	}
	// Closing stdin tells omnitun-tor to exit (also when this process dies).
	if p.stdin, err = p.cmd.StdinPipe(); err != nil {
		p.closeTransports()
		return nil, err
	}
	stderr, _ := p.cmd.StderrPipe()
	if err := p.cmd.Start(); err != nil {
		p.closeTransports()
		return nil, fmt.Errorf("start Tor: %w", err)
	}

	listening := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			var ev event
			if json.Unmarshal(sc.Bytes(), &ev) != nil {
				continue
			}
			switch ev.Event {
			case "listening":
				listening <- ev.Socks
			case "bootstrap":
				if cfg.OnBootstrap != nil {
					status := ev.Status
					if ev.Blocked != "" {
						status += " (" + ev.Blocked + ")"
					}
					cfg.OnBootstrap(ev.Percent, status)
				}
			case "ready":
				p.ready <- nil
			case "error":
				p.ready <- errors.New(ev.Message)
			}
		}
	}()
	if stderr != nil {
		go func() {
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				if cfg.Logf != nil {
					cfg.Logf("tor: %s", sc.Text())
				}
			}
		}()
	}
	go func() {
		err := p.cmd.Wait()
		p.err = err
		select {
		case p.ready <- fmt.Errorf("Tor exited: %v", err):
		default:
		}
		close(p.done)
	}()

	select {
	case addr := <-listening:
		p.SocksAddr = addr
		return p, nil
	case <-p.done:
		p.closeTransports()
		return nil, fmt.Errorf("Tor exited at start: %v", p.err)
	}
}

// Ready waits for the bootstrap to finish.
func (p *Process) Ready(ctx context.Context) error {
	select {
	case err := <-p.ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Done is closed when the Tor process exits.
func (p *Process) Done() <-chan struct{} { return p.done }

// Stop ends Tor and its transports.
func (p *Process) Stop() {
	p.once.Do(func() {
		p.stdin.Close()
		if p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}
		<-p.done
		p.closeTransports()
	})
}

func (p *Process) closeTransports() {
	if p.pts != nil {
		p.pts.Close()
	}
}

// Port returns the SOCKS port number.
func (p *Process) Port() int {
	_, port, _ := cutLast(p.SocksAddr, ":")
	n, _ := strconv.Atoi(port)
	return n
}

func cutLast(s, sep string) (string, string, bool) {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
