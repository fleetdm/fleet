package scep

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	smallstepscep "github.com/smallstep/scep"
)

// EnrollmentClient implements fleet.SCEPEnrollmentClient.
type EnrollmentClient struct {
	logger  *slog.Logger
	timeout *time.Duration
	clock   clock.Clock

	caCertsMu sync.Mutex
	caCerts   map[string]cachedCACerts // by SCEP URL
}

type cachedCACerts struct {
	certs     []*x509.Certificate
	fetchedAt time.Time
}

var _ fleet.SCEPEnrollmentClient = (*EnrollmentClient)(nil)

// NewEnrollmentClient returns an EnrollmentClient whose SCEP requests time out after 30 seconds.
func NewEnrollmentClient(logger *slog.Logger) *EnrollmentClient {
	return &EnrollmentClient{
		logger:  logger,
		timeout: new(30 * time.Second),
		clock:   clock.C,
		caCerts: make(map[string]cachedCACerts),
	}
}

// GetCertificate enrolls csr against the SCEP server at url. The caller holds the CSR's private
// key, so the SCEP envelope is signed with an ephemeral key and self-signed certificate instead,
// which RFC 8894 section 2.3 allows. The CertRep is encrypted to that certificate.
func (c *EnrollmentClient) GetCertificate(ctx context.Context, url string, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	client, err := scepclient.New(url, c.logger, scepclient.WithTimeout(c.timeout))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	caCerts, err := c.cachedCACerts(ctx, client, url)
	if err != nil {
		return nil, err
	}
	cert, err := c.enroll(ctx, client, caCerts, csr)
	if err != nil {
		// A renewed RA certificate makes every request against the old chain fail, so any failure
		// refetches the chain on the next request instead of waiting out the TTL.
		c.caCertsMu.Lock()
		delete(c.caCerts, url)
		c.caCertsMu.Unlock()
		return nil, err
	}
	return cert, nil
}

// cachedCACerts returns url's GetCACert chain, fetching it when it is not cached or has expired.
// The lock is not held during the fetch, so a slow SCEP server does not block other requests.
func (c *EnrollmentClient) cachedCACerts(ctx context.Context, client scepclient.Client, url string) ([]*x509.Certificate, error) {
	c.caCertsMu.Lock()
	cached, ok := c.caCerts[url]
	c.caCertsMu.Unlock()
	if ok && c.clock.Since(cached.fetchedAt) < caCertsCacheTTL {
		return cached.certs, nil
	}

	certs, err := enrollment.FetchCACerts(ctx, client)
	if err != nil {
		return nil, wrapEnrollmentError(err)
	}
	c.caCertsMu.Lock()
	c.caCerts[url] = cachedCACerts{certs: certs, fetchedAt: c.clock.Now()}
	c.caCertsMu.Unlock()
	return certs, nil
}

// enroll sends csr to the SCEP server whose chain is caCerts and returns the issued certificate.
func (c *EnrollmentClient) enroll(ctx context.Context, client scepclient.Client, caCerts []*x509.Certificate, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	signerKey, signerCert, err := enrollment.NewEphemeralSigner(csr.Subject)
	if err != nil {
		return nil, err
	}
	cert, err := enrollment.Enroll(ctx, client, caCerts, enrollment.Request{
		CSR:        csr,
		SignerKey:  signerKey,
		SignerCert: signerCert,
		Logger:     c.logger,
	})
	if err != nil {
		return nil, wrapEnrollmentError(err)
	}
	return cert, nil
}

// caCertsCacheTTL is how long a SCEP URL's GetCACert chain is reused.
const caCertsCacheTTL = 10 * time.Minute

// enrollmentRetryAfterSeconds is the Retry-After for a transient SCEP request failure.
const enrollmentRetryAfterSeconds = 30

// wrapEnrollmentError marks a GetCACert or PKIOperation failure that should clear on its own as
// transient, and adds the challenge hint to a FAILURE. A transient error's cause is kept as text
// because wrapping a *net.OpError makes the encoder answer 408, and the response shows only this
// error's message. Other errors are returned as they are; the caller wraps them.
func wrapEnrollmentError(err error) error {
	if rejected, ok := errors.AsType[enrollment.RejectedError](err); ok && rejected.Status == smallstepscep.FAILURE {
		// Not every SCEP CA requires a challenge, so Fleet cannot check for one up front, but a
		// missing or wrong one is the likeliest cause of a rejection.
		return fmt.Errorf("%w; if this certificate authority requires a challenge, "+
			"include it as the CSR's challengePassword attribute", err)
	}
	reqErr, ok := errors.AsType[enrollment.RequestError](err)
	if !ok {
		return err
	}
	transient := !errors.Is(err, context.Canceled)
	if statusErr, ok := errors.AsType[scepserver.ResponseStatusError](err); ok {
		transient = statusErr.Code >= http.StatusInternalServerError ||
			statusErr.Code == http.StatusRequestTimeout || statusErr.Code == http.StatusTooManyRequests
	}
	if !transient {
		return err
	}
	return fleet.CertificateAuthorityTransientError{
		Message:           reqErr.Error(),
		RetryAfterSeconds: enrollmentRetryAfterSeconds,
	}
}
