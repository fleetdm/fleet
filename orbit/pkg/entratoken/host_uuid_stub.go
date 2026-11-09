//go:build !darwin || !cgo

package entratoken

import "errors"

func hostUUID() (string, error) {
	return "", errors.New("host UUID is only supported on macOS")
}
