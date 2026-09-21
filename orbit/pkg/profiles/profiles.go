package profiles

import "errors"

var (
	ErrNotFound       = errors.New("profile not found")
	ErrNotImplemented = errors.New("not implemented on this platform")

	// ErrEnrollSecretNotFound reports that no enroll secret is waiting for this device. It is the
	// ordinary state on a host that has already adopted one, so callers should treat it as "nothing
	// to do", not as a failure. Kept distinct from ErrNotFound, which is about the profile itself.
	ErrEnrollSecretNotFound = errors.New("no MDM-delivered enroll secret found")
)
