// Command omnitun-svc is the core daemon the UI talks to. It owns the tunnel
// and guarantees it never outlives the UI: the tunnel stops when no UI is
// connected, and the daemon exits when its parent process dies.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/NoobforAl/omnitun/core/control"
)

func defaultDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "omnitun")
}

func main() {
	listen := flag.String("listen", "127.0.0.1:0", "control address (loopback only)")
	controlFile := flag.String("control-file", filepath.Join(defaultDir(), "control.json"), "where to write the address and token")
	dataDir := flag.String("data-dir", defaultDir(), "engine data directory")
	parentPID := flag.Int("parent-pid", 0, "exit when this process exits (0 = don't watch)")
	grace := flag.Duration("grace", 5*time.Second, "how long the tunnel survives without a connected UI")
	flag.Parse()

	logger := log.New(os.Stderr, "omnitun-svc: ", log.LstdFlags)
	srv, err := control.NewServer(control.ServerConfig{
		DataDir: *dataDir,
		Grace:   *grace,
		Logf:    logger.Printf,
	})
	if err != nil {
		logger.Fatal(err)
	}
	addr, err := srv.Listen(*listen)
	if err != nil {
		logger.Fatal(err)
	}
	if err := srv.WriteControlFile(*controlFile); err != nil {
		logger.Fatal(err)
	}
	defer os.Remove(*controlFile)
	// The UI reads this line to learn the address when it launched us.
	fmt.Printf("READY %s\n", addr)

	exit := make(chan string, 2)
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		exit <- (<-sig).String()
	}()
	if *parentPID > 0 {
		go func() {
			if err := waitProcess(*parentPID); err != nil {
				exit <- fmt.Sprintf("cannot watch parent %d: %v", *parentPID, err)
				return
			}
			exit <- fmt.Sprintf("parent %d exited", *parentPID)
		}()
	}
	go func() {
		if err := srv.Serve(); err != nil {
			exit <- err.Error()
		}
	}()

	logger.Printf("listening on %s, shutting down: %s", addr, <-exit)
	srv.Close()
}
