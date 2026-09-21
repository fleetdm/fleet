// Package mdmsecret reads the enroll secret that Fleet MDM delivered to this device.
//
// On Windows the secret arrives in the registry, written by a Fleet-managed configuration
// profile rather than as an MSI property, so it is never visible to process listing at install
// time and never comes to rest in an MSI log or in secret.txt. orbit adopts the value and then
// clears it, so the presence of the value means "a secret is waiting" and its absence means
// orbit has already taken it. That makes the same channel serve both first enrollment and
// recovery: an administrator resends the profile and the value reappears.
//
// Only Windows has an implementation. macOS reads its equivalent from a configuration profile
// via orbit/pkg/profiles, and other platforms have no MDM channel to read from.
package mdmsecret

import "errors"

// ErrNotFound reports that no secret is waiting for this device. It is the ordinary state on a
// host that has already enrolled, so callers should treat it as "keep waiting", not as a failure.
var ErrNotFound = errors.New("no MDM-delivered enroll secret found")

// ErrNotImplemented reports that this platform has no MDM-delivered secret channel.
var ErrNotImplemented = errors.New("not implemented on this platform")
