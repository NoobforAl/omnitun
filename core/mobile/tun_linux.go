package mobile

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// tunName asks the kernel for the TUN interface name behind fd.
func tunName(fd int32) (string, error) {
	var ifr [unix.IFNAMSIZ + 64]byte
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TUNGETIFF), uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return "", fmt.Errorf("query tun name: %w", errno)
	}
	return unix.ByteSliceToString(ifr[:]), nil
}

func dupFD(fd int) (int, error) { return unix.Dup(fd) }
