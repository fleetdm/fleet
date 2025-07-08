package hostidscep

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/smallstep/scep"
)

const (
	scepPath         = "/api/fleet/orbit/host_identity/scep"
	scepValidityDays = 365
)

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

// challengeMiddleware checks that ChallengePassword matches an enrollment secret
func challengeMiddleware(ds fleet.Datastore, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
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
	// [ ] Renewal            // Supports certificate renewal (same key, new cert)
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
		err = errors.New("no signed certificate")
	}
	if err != nil {
		svc.logger.Log("msg", "failed to sign CSR", "err", err)
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
