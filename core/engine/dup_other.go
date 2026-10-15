//go:build !unix

package engine

import "errors"

func dupFD(int) (int, error) {
	return -1, errors.New("TUN descriptors are only supported on Android/Linux")
}
