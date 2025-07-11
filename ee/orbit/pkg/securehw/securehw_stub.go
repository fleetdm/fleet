//go:build !linux
// +build !linux

package securehw

import (
	"github.com/rs/zerolog"
)

func newTEE(string, zerolog.Logger) (TEE, error) {
	return nil, nil
}
