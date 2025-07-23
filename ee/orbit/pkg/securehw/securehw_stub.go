//go:build !linux
// +build !linux

package securehw

import (
	"errors"

	"github.com/rs/zerolog"
)

func newSecureHW(string, zerolog.Logger) (SecureHW, error) {
	return nil, errors.New("not implemented")
}
