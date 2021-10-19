//go:build windows
// +build windows

package secure

import "os"

func MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}
