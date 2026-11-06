//go:build !darwin

package entratoken

import (
	"context"
	"errors"
)

// GetIDTokenAsLoggedInUser is only supported on macOS.
func GetIDTokenAsLoggedInUser(context.Context, Request) (string, error) {
	return "", errors.New("Entra device token is only supported on macOS")
}
