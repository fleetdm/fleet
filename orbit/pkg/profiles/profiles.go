package profiles

import "errors"

var (
	ErrNotFound       = errors.New("profile not found")
	ErrNotImplemented = errors.New("not implemented on this platform")

	// ErrEnrollSecretNotFound reports that no enroll secret is waiting for this device.
	ErrEnrollSecretNotFound = errors.New("no MDM-delivered enroll secret found")
)
