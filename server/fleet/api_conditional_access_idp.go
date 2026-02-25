package fleet

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
)

type ConditionalAccessGetIdPSigningCertRequest struct{}

type ConditionalAccessGetIdPSigningCertResponse struct {
	CertPEM []byte
	Err     error `json:"error,omitempty"`
}

func (r ConditionalAccessGetIdPSigningCertResponse) Error() error { return r.Err }

func (r ConditionalAccessGetIdPSigningCertResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.CertPEM)), 10))
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fleet-idp-signing-cert.pem\"")

	// OK to just log the error here as writing anything on `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the header provided
	n, err := w.Write(r.CertPEM)
	if err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_written", n)
	}
}

type ConditionalAccessGetIdPAppleProfileResponse struct {
	ProfileData []byte
	Err         error `json:"error,omitempty"`
}

func (r ConditionalAccessGetIdPAppleProfileResponse) Error() error { return r.Err }

func (r ConditionalAccessGetIdPAppleProfileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.ProfileData)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fleet-conditional-access.mobileconfig\"")

	// OK to just log the error here as writing anything on `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the header provided
	n, err := w.Write(r.ProfileData)
	if err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_written", n)
	}
}
