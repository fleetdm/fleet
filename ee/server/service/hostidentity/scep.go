package hostidentity

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/smallstep/scep"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	scepPath                    = "/api/fleet/orbit/host_identity/scep"
	scepValidityDays            = 365
	renewalCertKey   contextKey = "renewal_cert"
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

// RegisterSCEP registers the HTTP handler for SCEP service needed for fleetd enrollment.
func RegisterSCEP(
	mux *http.ServeMux,
	scepStorage scepdepot.Depot,
	ds fleet.Datastore,
	logger kitlog.Logger,
) error {
	err := initAssets(ds)
	if err != nil {
		return fmt.Errorf("initializing host identity assets: %w", err)
	}
	var signer scepserver.CSRSignerContext = scepserver.SignCSRAdapter(scepdepot.NewSigner(
		scepStorage,
		scepdepot.WithValidityDays(scepValidityDays),
		scepdepot.WithAllowRenewalDays(scepValidityDays/2),
	))

	signer = challengeMiddleware(ds, signer)
	scepService := NewSCEPService(
		ds,
		signer,
		kitlog.With(logger, "component", "host-id-scep"),
	)

	scepLogger := kitlog.With(logger, "component", "http-host-id-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)

	// Note: Monitoring (APM/OpenTel) is missing for this SCEP server.
	// In addition, the scepserver error handler does not send errors to APM/Sentry/Redis.
	// It should be enhanced to do so if/when we start monitoring error traces.
	// This note also applies to the other SCEP servers we use.
	// That is why we're not using ctxerr wrappers here.
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle(scepPath, scepHandler)
	return nil
}

// challengeMiddleware checks authentication for SCEP requests.
// For initial enrollment: validates ChallengePassword against enrollment secrets
// For renewal: validates that the request is signed by an existing valid certificate
func challengeMiddleware(ds fleet.Datastore, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	// We need to wrap the middleware to have access to the full PKIMessage
	// The CSRSignerContext only receives the CSRReqMessage, not the full PKIMessage
	// So we'll check renewal at the PKIOperation level instead
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// Check if we have renewal info in context (set by PKIOperation)
		if renewalCert, ok := ctx.Value(renewalCertKey).(*x509.Certificate); ok && renewalCert != nil {
			// This is a renewal request
			// The SCEP library has already validated the signature using the signer certificate
			// We just need to verify that:
			// 1. The certificate is not expired
			// 2. The CSR has the same public key as the existing certificate

			// Check if the certificate is still valid
			now := time.Now()
			if now.Before(renewalCert.NotBefore) || time.Now().After(renewalCert.NotAfter) {
				return nil, errors.New("signer certificate is not valid")
			}

			// Verify that the CSR uses the same public key as the existing certificate
			keysEqual, err := PublicKeysEqual(m.CSR.PublicKey, renewalCert.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("error comparing public keys: %w", err)
			}
			if !keysEqual {
				return nil, errors.New("CSR public key does not match signer certificate")
			}

			// For renewal, we don't need a challenge password
			return next.SignCSRContext(ctx, m)
		}

		// This is an initial enrollment request - require challenge password
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
	// The (chainable) CSR signing function. Intended to handle all
	// SCEP request functionality such as CSR & challenge checking, CA
	// issuance, RA proxying, etc.
	signer scepserver.CSRSignerContext

	logger log.Logger

	ds fleet.MDMAssetRetriever
}

func (svc *service) GetCACaps(_ context.Context) ([]byte, error) {
	// Supported SCEP CA Capabilities:
	//
	// Cryptographic and Algorithm Support:
	// [x] POSTPKIOperation   // Supports HTTP POST for PKIOperation (preferred over GET)
	// [ ] SHA-1              // Supports SHA-1 for signing
	// [x] SHA-256            // Supports SHA-256 for signing
	// [ ] SHA-512            // Supports SHA-512 for signing
	// [x] AES                // Supports AES encryption for PKCS#7 enveloped data
	// [ ] DES3               // Supports Triple DES encryption - older, weaker encryption
	//
	// Operational Capabilities:
	// [ ] GetNextCACert      // Supports fetching next CA certificate (rollover)
	// [x] Renewal            // Supports certificate renewal (same key, new cert)
	// [ ] Update             // Supports certificate update (new key)
	//
	// These capabilities are implied by the protocol and don't need to be explicitly declared:
	// [x] SCEPStandard       // Conforms to a known SCEP standard version
	// [x] PKCS7              // Responses are in PKCS#7 format
	// [x] X509               // Supports X.509 certificates
	//
	defaultCaps := []byte("SHA-256\nAES\nPOSTPKIOperation\nRenewal")
	return defaultCaps, nil
}

func (svc *service) GetCACert(ctx context.Context, _ string) ([]byte, int, error) {
	cert, err := caKeyPair(ctx, svc.ds)
	if err != nil {
		return nil, 0, fmt.Errorf("retrieving host identity SCEP CA certificate (GetCACert): %w", err)
	}
	return cert.Leaf.Raw, 1, nil
}

func caKeyPair(ctx context.Context, ds fleet.MDMAssetRetriever) (*tls.Certificate, error) {
	return assets.KeyPair(ctx, ds, fleet.MDMAssetHostIdentityCACert, fleet.MDMAssetHostIdentityCAKey)
}

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
		return nil, fmt.Errorf("retrieving host identity SCEP CA certificate: %w", err)
	}

	pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key not in RSA format")
	}

	if err := msg.DecryptPKIEnvelope(cert.Leaf, pk); err != nil {
		return nil, err
	}

	// Check if this is a renewal request by looking at the SignerCert
	if msg.SignerCert != nil {
		// Pass the signer certificate through context for the challengeMiddleware
		ctx = context.WithValue(ctx, renewalCertKey, msg.SignerCert)
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

func (svc *service) GetNextCACert(_ context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// NewSCEPService creates a new scep service
func NewSCEPService(ds fleet.Datastore, signer scepserver.CSRSignerContext, logger log.Logger) scepserver.Service {
	return &service{
		ds:     ds,
		signer: signer,
		logger: logger,
	}
}

func PublicKeysEqual(a, b crypto.PublicKey) (bool, error) {
	derA, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false, fmt.Errorf("marshal a: %w", err)
	}
	derB, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false, fmt.Errorf("marshal b: %w", err)
	}
	return bytes.Equal(derA, derB), nil
}
