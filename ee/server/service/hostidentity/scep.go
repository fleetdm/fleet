package hostidentity

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/smallstep/scep"
)

const (
	scepPath         = "/api/fleet/orbit/host_identity/scep"
	scepValidityDays = 365
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

// getCertValidityDays returns the certificate validity period in days.
// It checks for FLEET_DEV_HOST_IDENTITY_CERT_VALIDITY_DAYS environment variable
// and falls back to scepValidityDays if not set or invalid.
func getCertValidityDays() int {
	if envValue := os.Getenv("FLEET_DEV_HOST_IDENTITY_CERT_VALIDITY_DAYS"); envValue != "" {
		if days, err := strconv.Atoi(envValue); err == nil && days > 0 {
			return days
		}
	}
	return scepValidityDays
}

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
		scepdepot.WithValidityDays(getCertValidityDays()),
	))

	signer = challengeMiddleware(ds, signer)
	signer = renewalMiddleware(ds, logger, signer)
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

// challengeMiddleware checks that ChallengePassword matches an enrollment secret
func challengeMiddleware(ds fleet.Datastore, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// Check if this is a renewal request by looking for the custom Fleet extension
		if hasRenewalExtension(m.CSR) {
			// Skip challenge verification for renewal requests
			// The renewal middleware will handle authentication
			return next.SignCSRContext(ctx, m)
		}

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

// hasRenewalExtension checks if the CSR contains the renewal extension
func hasRenewalExtension(csr *x509.CertificateRequest) bool {
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(types.RenewalExtensionOID) {
			return true
		}
	}
	return false
}

// renewalMiddleware handles certificate renewal with proof-of-possession
func renewalMiddleware(ds fleet.Datastore, logger kitlog.Logger, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// Check if this is a renewal request
		var renewalData types.RenewalData
		found := false
		for _, ext := range m.CSR.Extensions {
			if ext.Id.Equal(types.RenewalExtensionOID) {
				if err := json.Unmarshal(ext.Value, &renewalData); err != nil {
					return nil, fmt.Errorf("invalid renewal extension: %w", err)
				}
				found = true
				break
			}
		}

		if !found {
			// Not a renewal request, pass through
			return next.SignCSRContext(ctx, m)
		}

		logger.Log("msg", "processing renewal request", "serial", renewalData.SerialNumber)

		// Parse the serial number from hex
		serialBigInt := new(big.Int)
		_, success := serialBigInt.SetString(strings.TrimPrefix(renewalData.SerialNumber, "0x"), 16)
		if !success {
			return nil, fmt.Errorf("invalid serial number format: %s", renewalData.SerialNumber)
		}

		// Retrieve the old certificate data
		oldCertData, err := ds.GetHostIdentityCertBySerialNumber(ctx, serialBigInt.Uint64())
		if err != nil {
			return nil, fmt.Errorf("retrieving old certificate: %w", err)
		}

		// Get the public key from the stored data
		pubKey, err := oldCertData.UnmarshalPublicKey()
		if err != nil {
			return nil, fmt.Errorf("unmarshaling public key: %w", err)
		}

		// Verify the signature
		sigBytes, err := base64.StdEncoding.DecodeString(renewalData.Signature)
		if err != nil {
			return nil, fmt.Errorf("decoding signature: %w", err)
		}

		// Verify the signature
		hash := sha256.Sum256([]byte(renewalData.SerialNumber))
		if !ecdsa.VerifyASN1(pubKey, hash[:], sigBytes) {
			return nil, errors.New("invalid renewal signature")
		}

		logger.Log("msg", "renewal signature verified", "serial", renewalData.SerialNumber, "cn", oldCertData.CommonName)

		// Issue the new certificate
		newCert, err := next.SignCSRContext(ctx, m)
		if err != nil {
			return nil, fmt.Errorf("signing renewal CSR: %w", err)
		}

		// Update the new certificate's host_id to match the old certificate
		if oldCertData.HostID != nil {
			err = ds.UpdateHostIdentityCertHostIDBySerial(ctx, newCert.SerialNumber.Uint64(), *oldCertData.HostID)
			if err != nil {
				// Log the error but don't fail the renewal
				ctxerr.Handle(ctx, err)
				level.Error(logger).Log("msg", "failed to update host_id for renewed certificate", "err", err, "new_serial",
					newCert.SerialNumber.Uint64(), "host_id", *oldCertData.HostID)
			}
		}

		return newCert, nil
	}
}

var _ scepserver.Service = (*service)(nil)

type service struct {
	// The (chainable) CSR signing function. Intended to handle all
	// SCEP request functionality such as CSR & challenge checking, CA
	// issuance, RA proxying, etc.
	signer scepserver.CSRSignerContext

	logger log.Logger

	ds fleet.Datastore
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
	// [ ] Renewal            // Supports certificate renewal (same or new key, new cert)
	// [ ] Update             // Supports certificate update (new key)
	//
	// These capabilities are implied by the protocol and don't need to be explicitly declared:
	// [x] SCEPStandard       // Conforms to a known SCEP standard version
	// [x] PKCS7              // Responses are in PKCS#7 format
	// [x] X509               // Supports X.509 certificates
	//
	defaultCaps := []byte("SHA-256\nAES\nPOSTPKIOperation")
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
