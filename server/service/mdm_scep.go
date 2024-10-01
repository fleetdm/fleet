package service

import (
	"context"
	"crypto/rsa"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/go-kit/log"
	"github.com/smallstep/scep"
)

var _ scepserver.Service = (*service)(nil)

type service struct {
	// The (chainable) CSR signing function. Intended to handle all
	// SCEP request functionality such as CSR & challenge checking, CA
	// issuance, RA proxying, etc.
	signer scepserver.CSRSignerContext

	/// info logging is implemented in the service middleware layer.
	debugLogger log.Logger

	ds fleet.MDMAssetRetriever
}

func (svc *service) GetCACaps(ctx context.Context) ([]byte, error) {
	defaultCaps := []byte("Renewal\nSHA-1\nSHA-256\nAES\nDES3\nSCEPStandard\nPOSTPKIOperation")
	return defaultCaps, nil
}

func (svc *service) GetCACert(ctx context.Context, _ string) ([]byte, int, error) {
	cert, err := assets.CAKeyPair(ctx, svc.ds)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "parsing SCEP certificate")
	}
	return cert.Leaf.Raw, 1, nil
}

func (svc *service) PKIOperation(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, &fleet.BadRequestError{Message: "missing data for PKIOperation"}
	}
	msg, err := scep.ParsePKIMessage(data, scep.WithLogger(svc.debugLogger))
	if err != nil {
		return nil, err
	}

	cert, err := assets.CAKeyPair(ctx, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing SCEP certificate")
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
		svc.debugLogger.Log("msg", "failed to sign CSR", "err", err)
		certRep, err := msg.Fail(cert.Leaf, pk, scep.BadRequest)
		return certRep.Raw, err
	}

	certRep, err := msg.Success(cert.Leaf, pk, crt)
	return certRep.Raw, err
}

func (svc *service) GetNextCACert(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// NewService creates a new scep service
func NewSCEPService(ds fleet.MDMAssetRetriever, signer scepserver.CSRSignerContext, logger log.Logger) scepserver.Service {
	return &service{
		signer:      signer,
		debugLogger: log.NewNopLogger(),
		ds:          ds,
	}
}
