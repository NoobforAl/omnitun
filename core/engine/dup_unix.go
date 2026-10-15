//go:build unix

package engine

import "syscall"

func dupFD(fd int) (int, error) { return syscall.Dup(fd) }
