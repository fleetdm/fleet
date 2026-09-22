//go:build !windows

package profiles

import "context"

// GetEnrollSecret is not implemented outside Windows.
func GetEnrollSecret() (string, error) {
	return "", ErrNotImplemented
}

// ClearEnrollSecret is not implemented outside Windows.
func ClearEnrollSecret() error {
	return ErrNotImplemented
}

// EnrollSecretWatch has no non-Windows implementations.
type EnrollSecretWatch struct{}

// Wait is not implemented outside Windows.
func (w *EnrollSecretWatch) Wait(_ context.Context) error { return ErrNotImplemented }

// Close is not implemented outside Windows.
func (w *EnrollSecretWatch) Close() error { return ErrNotImplemented }

// ArmEnrollSecretWatch is not implemented outside Windows.
func ArmEnrollSecretWatch() (*EnrollSecretWatch, error) {
	return nil, ErrNotImplemented
}

// EnsureEnrollSecretKey is not implemented outside Windows.
func EnsureEnrollSecretKey() error {
	return ErrNotImplemented
}
