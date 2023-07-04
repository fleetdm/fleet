package profiles

import "errors"

var (
	ErrNotFound       = errors.New("profile not found")
	ErrNotImplemented = errors.New("not implemented on this platform")
)
