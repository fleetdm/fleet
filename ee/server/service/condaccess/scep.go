package condaccess

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/smallstep/scep"
)

const (
	scepPath         = "/api/fleet/conditional_access/scep"
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
	logger kitlog.Logger,
	fleetConfig *config.FleetConfig,
) error {
	if fleetConfig == nil {
		return errors.New("fleet config is nil")
	}
	err := initAssets(ctx, ds)
	if err != nil {
		return fmt.Errorf("initializing conditional access assets: %w", err)
	}

	// Create signer without renewal middleware (key difference from host identity SCEP)
	var signer scepserver.CSRSignerContext = scepserver.SignCSRAdapter(scepdepot.NewSigner(
		scepStorage,
		scepdepot.WithValidityDays(scepValidityDays),
	))

	// Add challenge middleware for enrollment secret verification
	signer = challengeMiddleware(ds, signer)

	scepService := NewSCEPService(
		ds,
		signer,
		kitlog.With(logger, "component", "conditional-access-scep"),
	)

	scepLogger := kitlog.With(logger, "component", "http-conditional-access-scep")
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
	return nil
}

// challengeMiddleware checks that ChallengePassword matches an enrollment secret.
// Unlike host identity SCEP, this does NOT support renewal (no renewal extension check).
func challengeMiddleware(ds fleet.Datastore, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// No renewal support for conditional access SCEP
		// Always require a valid challenge password

		if m.ChallengePassword == "" {
			return nil, errors.New("missing challenge")
		}
		_, err := ds.VerifyEnrollSecret(ctx, m.ChallengePassword)
		switch {
		case fleet.IsNotFound(err):
			return nil, errors.New("invalid challenge")
		case err != nil:
			return nil, fmt.Errorf("verifying enrollment secret: %w", err)
		}
		return next.SignCSRContext(ctx, m)
	}
}

var _ scepserver.Service = (*service)(nil)

type service struct {
	// The (chainable) CSR signing function
	signer scepserver.CSRSignerContext
	logger log.Logger
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
	msg, err := scep.ParsePKIMessage(data, scep.WithLogger(svc.logger))
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
		return nil, err
	}

	crt, err := svc.signer.SignCSRContext(ctx, msg.CSRReqMessage)
	if err == nil && crt == nil {
		err = errors.New("signer returned nil certificate without error")
	}
	if err != nil {
		svc.logger.Log("msg", "failed to sign CSR", "err", err)

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
func NewSCEPService(ds fleet.Datastore, signer scepserver.CSRSignerContext, logger log.Logger) scepserver.Service {
	return &service{
		ds:     ds,
		signer: signer,
		logger: logger,
	}
}
