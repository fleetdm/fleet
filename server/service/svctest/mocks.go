package svctest

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
)

// mockMailService is the no-op mailer used by the test service.
type mockMailService struct {
	SendEmailFn func(e fleet.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(_ context.Context, e fleet.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}

func (svc *mockMailService) CanSendEmail(smtpSettings fleet.SMTPSettings) bool {
	return smtpSettings.SMTPConfigured
}

// nopEnrollHostLimiter is a no-op fleet.EnrollHostLimiter for tests.
type nopEnrollHostLimiter struct{}

func (nopEnrollHostLimiter) CanEnrollNewHost(_ context.Context) (bool, error) {
	return true, nil
}

func (nopEnrollHostLimiter) SyncEnrolledHostIDs(_ context.Context) error {
	return nil
}

// acmeCSRSigner adapts a depot.Signer to the acme.CSRSigner interface.
type acmeCSRSigner struct {
	signer *depot.Signer
}

func (a *acmeCSRSigner) SignCSR(_ context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	return a.signer.Signx509CSR(csr)
}

// mockRoundTripper is a custom http.RoundTripper that redirects requests to a
// mock server. It is exported here for tests that need to swap an external
// HTTP client to point at an httptest server.
type mockRoundTripper struct {
	mockServer  string
	origBaseURL string
	next        http.RoundTripper
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), rt.origBaseURL) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		newURL := fmt.Sprintf("%s/%s", rt.mockServer, path)
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body) //nolint:gosec // test helper
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return rt.next.RoundTrip(newReq)
	}
	return rt.next.RoundTrip(req)
}
