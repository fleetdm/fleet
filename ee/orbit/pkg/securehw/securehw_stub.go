//go:build !linux
// +build !linux

package securehw

import (
	"errors"

	"github.com/rs/zerolog"
)

func newTEE(string, zerolog.Logger) (TEE, error) {
	return nil, errors.New("not implemented")
}
