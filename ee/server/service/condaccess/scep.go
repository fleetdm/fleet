package condaccess

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/cenkalti/backoff/v4"
	condaccessdepot "github.com/fleetdm/fleet/v4/ee/server/service/condaccess/depot"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/kitlogadapter"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/smallstep/scep"
)

const (
	scepPath         = fleet.ConditionalAccessSCEPPath
	scepValidityDays = 398 // Apple's maximum allowed certificate validity
)

// RateLimitError is an error type that generates a 429 status code.
type RateLimitError struct {
	Message string
}

// Error returns the error message.
func (e *RateLimitError) Error() string {
	return e.Message
}

// StatusCode implements the kithttp StatusCoder interface
func (e *RateLimitError) StatusCode() int { return http.StatusTooManyRequests }

// RegisterSCEP registers the HTTP handler for conditional access SCEP service.
func RegisterSCEP(
	ctx context.Context,
	mux *http.ServeMux,
	scepStorage scepdepot.Depot,
	ds fleet.Datastore,
	logger *slog.Logger,
	fleetConfig *config.FleetConfig,
) error {
	if fleetConfig == nil {
		return errors.New("fleet config is nil")
	}
	err := initAssets(ctx, ds)
	if err != nil {
		return fmt.Errorf("initializing conditional access assets: %w", err)
	}

	scepServiceLogger := logger.With("component", "conditional-access-scep")
	// Create signer without renewal middleware (key difference from host identity SCEP)
	signer := challengeMiddleware(ds, scepServiceLogger, fleetConfig.MDM, scepdepot.NewSigner(
		scepStorage,
		scepdepot.WithValidityDays(scepValidityDays),
	))

	scepService := NewSCEPService(
		ds,
		signer,
		scepServiceLogger,
	)

	scepLogger := logger.With("component", "http-conditional-access-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)

	// The scepserver error handler does not send errors to APM/Sentry/Redis.
	// It should be enhanced to do so if/when we start monitoring error traces.
	// This note also applies to the other SCEP servers we use.
	// That is why we're not using ctxerr wrappers here.
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	scepHandler = otel.WrapHandler(scepHandler, scepPath, *fleetConfig)
	mux.Handle(scepPath, scepHandler)
	// The Windows SCEP client appends /pkiclient.exe to the server URL, as NDES expects, and can't be told not to.
	mux.Handle(scepPath+"/pkiclient.exe", scepHandler)
	return nil
}

type csrCallbackSigner interface {
	SignX509CSRWithCallback(csr *x509.CertificateRequest, subject pkix.Name, callback func(tmpl *x509.Certificate)) (*x509.Certificate, error)
}

// challengeMiddleware checks that ChallengePassword is a one-time challenge Fleet created for the host in the CSR's
// device URI, or a global enroll secret, which profiles written before one-time challenges carry, and signs the CSR
// with that device URI as the certificate's only SAN, rather than every SAN the requester asked for. The global enroll
// secret isn't accepted for hosts whose platform gets one-time enroll secrets.
// Unlike host identity SCEP, this does NOT support renewal (no renewal extension check).
func challengeMiddleware(ds fleet.Datastore, logger *slog.Logger, mdmConfig config.MDMConfig, next csrCallbackSigner) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		if m.ChallengePassword == "" {
			return nil, errors.New("missing challenge")
		}
		uri, hostUUID := condaccessdepot.DeviceURI(m.CSR.URIs)
		if hostUUID == "" {
			return nil, errors.New("no device UUID found in CSR SAN URI")
		}
		// Looked up before any challenge is used, so a request for a host that doesn't exist doesn't use one up.
		host, err := ds.HostByUUID(ctx, hostUUID)
		switch {
		case fleet.IsNotFound(err):
			return nil, errors.New("unknown host")
		case err != nil:
			return nil, fmt.Errorf("looking up host: %w", err)
		}

		sign := func() (*x509.Certificate, error) {
			return next.SignX509CSRWithCallback(m.CSR, m.CSR.Subject, func(tmpl *x509.Certificate) {
				tmpl.URIs = []*url.URL{uri}
			})
		}

		if mdmConfig.GlobalEnrollSecretAllowed(host.Platform) {
			secret, err := ds.VerifyEnrollSecret(ctx, m.ChallengePassword)
			switch {
			case err == nil:
				// Only global enroll secrets (team_id IS NULL) are valid for
				// conditional-access SCEP. Reject team-scoped secrets.
				if secret.TeamID != nil {
					return nil, errors.New("invalid challenge")
				}
				// Anyone holding the secret can get a certificate for any host, so admins need to find who still uses it.
				logger.WarnContext(ctx, "conditional access certificate requested with a global enroll secret, redeploy the Okta profile to use one-time challenges",
					"host_uuid", hostUUID)
				return sign()
			case !fleet.IsNotFound(err):
				return nil, fmt.Errorf("verifying enrollment secret: %w", err)
			}
		}

		err = ds.ConsumeConditionalAccessSCEPChallenge(ctx, m.ChallengePassword, hostUUID)
		switch {
		case fleet.IsNotFound(err):
			return nil, errors.New("invalid challenge")
		case err != nil:
			return nil, fmt.Errorf("consuming conditional access SCEP challenge: %w", err)
		}
		return sign()
	}
}

var _ scepserver.Service = (*service)(nil)

type service struct {
	// The (chainable) CSR signing function
	signer scepserver.CSRSignerContext
	logger *slog.Logger
	ds     fleet.Datastore
}

// GetCACaps returns the CA capabilities.
// Conditional access SCEP does NOT support renewal, so "Renewal" is excluded.
func (svc *service) GetCACaps(_ context.Context) ([]byte, error) {
	// Supported SCEP CA Capabilities:
	//
	// Cryptographic and Algorithm Support:
	// [x] POSTPKIOperation   // Supports HTTP POST for PKIOperation (preferred over GET)
	// [ ] SHA-1              // Not supported - deprecated
	// [x] SHA-256            // Supports SHA-256 for signing
	// [ ] SHA-512            // Not needed for our use case
	// [x] AES                // Supports AES encryption for PKCS#7 enveloped data
	// [ ] DES3               // Not supported - deprecated
	//
	// Operational Capabilities:
	// [ ] GetNextCACert      // Not supported - no CA rollover
	// [ ] Renewal            // NOT SUPPORTED - key difference from host identity SCEP
	// [ ] Update             // Not supported
	//
	// Note: Renewal is explicitly excluded for conditional access SCEP.
	// Clients must re-enroll with a valid challenge to get a new certificate.
	defaultCaps := []byte("SHA-256\nAES\nPOSTPKIOperation")
	return defaultCaps, nil
}

// GetCACert returns the CA certificate.
func (svc *service) GetCACert(ctx context.Context, _ string) ([]byte, int, error) {
	cert, err := caKeyPair(ctx, svc.ds)
	if err != nil {
		return nil, 0, fmt.Errorf("retrieving conditional access SCEP CA certificate (GetCACert): %w", err)
	}
	return cert.Leaf.Raw, 1, nil
}

// caKeyPair retrieves the CA certificate and key for conditional access.
func caKeyPair(ctx context.Context, ds fleet.MDMAssetRetriever) (*tls.Certificate, error) {
	return assets.KeyPair(ctx, ds, fleet.MDMAssetConditionalAccessCACert, fleet.MDMAssetConditionalAccessCAKey)
}

// PKIOperation handles SCEP PKI operations (certificate signing).
func (svc *service) PKIOperation(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, &fleet.BadRequestError{Message: "missing data for PKIOperation"}
	}

	if err := cryptoutil.ValidateBERDepth(data, cryptoutil.MaxBERDepth); err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "invalid request body",
			InternalErr: err,
		}
	}

	msg, err := scep.ParsePKIMessage(data, scep.WithLogger(kitlogadapter.NewLogger(svc.logger)))
	if err != nil {
		return nil, err
	}

	cert, err := caKeyPair(ctx, svc.ds)
	if err != nil {
		return nil, fmt.Errorf("retrieving conditional access SCEP CA certificate: %w", err)
	}

	pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key not in RSA format")
	}

	if err := msg.DecryptPKIEnvelope(cert.Leaf, pk); err != nil {
		svc.logger.ErrorContext(ctx, "failed to decrypt PKI envelope", "err", err)
		certRep, err := msg.Fail(cert.Leaf, pk, scep.BadRequest)
		if certRep == nil {
			return nil, err
		}
		return certRep.Raw, err
	}

	crt, err := svc.signer.SignCSRContext(ctx, msg.CSRReqMessage)
	if err == nil && crt == nil {
		err = errors.New("signer returned nil certificate without error")
	}
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to sign CSR", "err", err)

		// Check if this is a rate limit error (permanent error from backoff)
		var permanentErr *backoff.PermanentError
		if errors.As(err, &permanentErr) {
			// Return HTTP 429 for rate limit errors
			return nil, &RateLimitError{Message: err.Error()}
		}

		certRep, err := msg.Fail(cert.Leaf, pk, scep.BadRequest)
		if certRep == nil {
			return nil, err
		}
		return certRep.Raw, err
	}

	certRep, err := msg.Success(cert.Leaf, pk, crt)
	if certRep == nil {
		return nil, err
	}
	return certRep.Raw, err
}

// GetNextCACert is not implemented for conditional access SCEP.
func (svc *service) GetNextCACert(_ context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// NewSCEPService creates a new conditional access SCEP service.
func NewSCEPService(ds fleet.Datastore, signer scepserver.CSRSignerContext, logger *slog.Logger) scepserver.Service {
	return &service{
		ds:     ds,
		signer: signer,
		logger: logger,
	}
}
