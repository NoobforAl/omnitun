package main

import "golang.org/x/sys/windows"

// waitProcess blocks until the process exits.
func waitProcess(pid int) error {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	_, err = windows.WaitForSingleObject(h, windows.INFINITE)
	return err
}
