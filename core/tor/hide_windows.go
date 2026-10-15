//go:build windows

package tor

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps the Tor console window hidden on Windows.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
