//go:build !darwin && !linux

package sshd_config

import (
	"errors"
	"io/fs"
)

func resolveTrusted(string) (string, fs.FileInfo, error) {
	return "", nil, errors.New("sshd_config is only supported on macOS and Linux")
}
