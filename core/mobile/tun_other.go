//go:build !linux

package mobile

// The mobile platform glue only runs on Android (linux); these stubs keep the
// package building elsewhere for tests and vet.

func tunName(int32) (string, error) { return "", errUnsupported }

func dupFD(int) (int, error) { return -1, errUnsupported }
