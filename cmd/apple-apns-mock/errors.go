package main

import "net/http"

// apnsError is one of APNs' rejection reasons. The reason string goes into the
// response body verbatim; see apnsPushError.
type apnsError struct {
	reason string
	status int
}

func (e *apnsError) Error() string { return e.reason }

// The reasons this mock returns, matching real APNs status/reason pairs.
var (
	errInvalidPushType    = &apnsError{"InvalidPushType", http.StatusBadRequest}
	errBadDeviceToken     = &apnsError{"BadDeviceToken", http.StatusBadRequest}
	errMissingDeviceToken = &apnsError{"MissingDeviceToken", http.StatusBadRequest}
	errBadExpirationDate  = &apnsError{"BadExpirationDate", http.StatusBadRequest}
	errPayloadEmpty       = &apnsError{"PayloadEmpty", http.StatusBadRequest}
	errBadMessageID       = &apnsError{"BadMessageId", http.StatusBadRequest}
	errPayloadTooLarge    = &apnsError{"PayloadTooLarge", http.StatusRequestEntityTooLarge}
	errInternalServer     = &apnsError{"InternalServerError", http.StatusInternalServerError}
	// What APNs returns when it cannot take a push right now; used when Redis
	// is unreachable.
	errServiceUnavailable = &apnsError{"ServiceUnavailable", http.StatusServiceUnavailable}
)
