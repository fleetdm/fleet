//go:build !windows

package mdmsecret

// Read is not implemented outside Windows. macOS reads its MDM-delivered enroll secret from a
// configuration profile via orbit/pkg/profiles instead.
func Read() (string, error) {
	return "", ErrNotImplemented
}

// Clear is not implemented outside Windows.
func Clear() error {
	return ErrNotImplemented
}
