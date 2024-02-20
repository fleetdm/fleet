package scepserver

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"

	"github.com/go-kit/kit/log"
)

// Service is the interface for all supported SCEP server operations.
type Service interface {
	// GetCACaps returns a list of options
	// which are supported by the server.
	GetCACaps(ctx context.Context) ([]byte, error)

	// GetCACert returns CA certificate or
	// a CA certificate chain with intermediates
	// in a PKCS#7 Degenerate Certificates format
	// message is an optional string for the CA
	GetCACert(ctx context.Context, message string) ([]byte, int, error)

	// PKIOperation handles incoming SCEP messages such as PKCSReq and
	// sends back a CertRep PKIMessag.
	PKIOperation(ctx context.Context, msg []byte) ([]byte, error)

	// GetNextCACert returns a replacement certificate or certificate chain
	// when the old one expires. The response format is a PKCS#7 Degenerate
	// Certificates type.
	GetNextCACert(ctx context.Context) ([]byte, error)
}

type service struct {
	// The service certificate and key for SCEP exchanges. These are
	// quite likely the same as the CA keypair but may be its own SCEP
	// specific keypair in the case of e.g. RA (proxy) operation.
	crt *x509.Certificate
	key *rsa.PrivateKey

	// Optional additional CA certificates for e.g. RA (proxy) use.
	// Only used in this service when responding to GetCACert.
	addlCa []*x509.Certificate

	// The (chainable) CSR signing function. Intended to handle all
	// SCEP request functionality such as CSR & challenge checking, CA
	// issuance, RA proxying, etc.
	signer CSRSigner

	/// info logging is implemented in the service middleware layer.
	debugLogger log.Logger
}

func (svc *service) GetCACaps(ctx context.Context) ([]byte, error) {
	defaultCaps := []byte("Renewal\nSHA-1\nSHA-256\nAES\nDES3\nSCEPStandard\nPOSTPKIOperation")
	return defaultCaps, nil
}

func (svc *service) GetCACert(ctx context.Context, _ string) ([]byte, int, error) {
	if svc.crt == nil {
		return nil, 0, errors.New("missing CA certificate")
	}
	if len(svc.addlCa) < 1 {
		return svc.crt.Raw, 1, nil
	}
	certs := []*x509.Certificate{svc.crt}
	certs = append(certs, svc.addlCa...)
	data, err := scep.DegenerateCertificates(certs)
	return data, len(svc.addlCa) + 1, err
}

func (svc *service) PKIOperation(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, &BadRequestError{Message: "missing data for PKIOperation"}
	}
	msg, err := scep.ParsePKIMessage(data, scep.WithLogger(svc.debugLogger))
	if err != nil {
		return nil, err
	}
	if err := msg.DecryptPKIEnvelope(svc.crt, svc.key); err != nil {
		return nil, err
	}

	crt, err := svc.signer.SignCSR(msg.CSRReqMessage)
	if err == nil && crt == nil {
		err = errors.New("no signed certificate")
	}
	if err != nil {
		svc.debugLogger.Log("msg", "failed to sign CSR", "err", err)
		certRep, err := msg.Fail(svc.crt, svc.key, scep.BadRequest)
		return certRep.Raw, err
	}

	certRep, err := msg.Success(svc.crt, svc.key, crt)
	return certRep.Raw, err
}

func (svc *service) GetNextCACert(ctx context.Context) ([]byte, error) {
	panic("not implemented")
}

// ServiceOption is a server configuration option
type ServiceOption func(*service) error

// WithLogger configures a logger for the SCEP Service.
// By default, a no-op logger is used.
func WithLogger(logger log.Logger) ServiceOption {
	return func(s *service) error {
		s.debugLogger = logger
		return nil
	}
}

// WithAddlCA appends an additional certificate to the slice of CA certs
func WithAddlCA(ca *x509.Certificate) ServiceOption {
	return func(s *service) error {
		s.addlCa = append(s.addlCa, ca)
		return nil
	}
}

// NewService creates a new scep service
func NewService(crt *x509.Certificate, key *rsa.PrivateKey, signer CSRSigner, opts ...ServiceOption) (Service, error) {
	s := &service{
		crt:         crt,
		key:         key,
		signer:      signer,
		debugLogger: log.NewNopLogger(),
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
