package service

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/go-kit/log"
)

var _ scepserver.Service = (*service)(nil)

type service struct {
	// The (chainable) CSR signing function. Intended to handle all
	// SCEP request functionality such as CSR & challenge checking, CA
	// issuance, RA proxying, etc.
	signer scepserver.CSRSigner

	/// info logging is implemented in the service middleware layer.
	debugLogger log.Logger

	ds fleet.MDMAssetRetriever
}

func (svc *service) GetCACaps(ctx context.Context) ([]byte, error) {
	defaultCaps := []byte("Renewal\nSHA-1\nSHA-256\nAES\nDES3\nSCEPStandard\nPOSTPKIOperation")
	return defaultCaps, nil
}

func (svc *service) GetCACert(ctx context.Context, _ string) ([]byte, int, error) {
	cert, _, err := svc.getKeypair(ctx)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "parsing SCEP certificate")
	}
	return cert.Raw, 1, nil
}

func (svc *service) PKIOperation(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, &fleet.BadRequestError{Message: "missing data for PKIOperation"}
	}
	msg, err := scep.ParsePKIMessage(data, scep.WithLogger(svc.debugLogger))
	if err != nil {
		return nil, err
	}

	cert, key, err := svc.getKeypair(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing SCEP certificate")
	}

	if err := msg.DecryptPKIEnvelope(cert, key); err != nil {
		return nil, err
	}

	crt, err := svc.signer.SignCSR(msg.CSRReqMessage)
	if err == nil && crt == nil {
		err = errors.New("no signed certificate")
	}
	if err != nil {
		svc.debugLogger.Log("msg", "failed to sign CSR", "err", err)
		certRep, err := msg.Fail(cert, key, scep.BadRequest)
		return certRep.Raw, err
	}

	certRep, err := msg.Success(cert, key, crt)
	return certRep.Raw, err
}

func (svc *service) GetNextCACert(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (svc *service) getKeypair(ctx context.Context) (*x509.Certificate, *rsa.PrivateKey, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	if err != nil {
		return nil, nil, fmt.Errorf("getting assets from database: %w", err)
	}

	cert, err := tls.X509KeyPair(assets[fleet.MDMAssetCACert].Value, assets[fleet.MDMAssetCAKey].Value)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing keypair: %w", err)
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse leaf certificate: %w", err)
	}

	pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("private key not in RSA format")
	}

	return parsed, pk, nil

}

// NewService creates a new scep service
func NewSCEPService(ds fleet.MDMAssetRetriever, signer scepserver.CSRSigner, logger log.Logger) scepserver.Service {
	return &service{
		signer:      signer,
		debugLogger: log.NewNopLogger(),
		ds:          ds,
	}
}
