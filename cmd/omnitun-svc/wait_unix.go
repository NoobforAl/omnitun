//go:build !windows

package main

import (
	"errors"
	"syscall"
	"time"
)

// waitProcess polls until the process exits.
func waitProcess(pid int) error {
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return err
	}
	for {
		time.Sleep(time.Second)
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return nil
		}
	}
}
