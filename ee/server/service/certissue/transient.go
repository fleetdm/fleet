package certissue

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// IsTransientStatus reports whether an HTTP status means that the CA can't serve the request now, but is expected to later.
func IsTransientStatus(status int) bool {
	return status >= http.StatusInternalServerError || status == http.StatusRequestTimeout || status == http.StatusTooManyRequests
}

// Transient marks err, the failure to get a response from a CA, as one that is expected to clear on its own, so it is worth retrying.
// It returns err as it is when retrying doesn't help: the request was canceled, or the TLS certificate of the CA isn't trusted.
func Transient(ctx context.Context, err error) error {
	var verifyErr *tls.CertificateVerificationError
	if ctx.Err() != nil || errors.As(err, &verifyErr) {
		return err
	}
	return fleet.CertificateAuthorityTransientError{Message: err.Error()}
}

// TransientStatus is Transient for an unsuccessful HTTP response: it marks err only when the status is one that IsTransientStatus accepts.
func TransientStatus(err error, status int) error {
	if !IsTransientStatus(status) {
		return err
	}
	return fleet.CertificateAuthorityTransientError{Message: err.Error()}
}
