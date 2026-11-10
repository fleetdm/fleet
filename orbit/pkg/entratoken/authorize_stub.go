//go:build !darwin || !cgo

package entratoken

import (
	"context"
	"errors"
)

func authorize(context.Context, string) (*authorizeResponse, error) {
	return nil, errors.New("Entra device token is only supported on macOS")
}
