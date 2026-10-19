package fleet

import (
	"context"
)

// DigiCertCertificate is the certificate DigiCert issued for a host.
type DigiCertCertificate = IssuedCertificate

type DigiCertService interface {
	VerifyProfileID(ctx context.Context, config DigiCertCA) error
	GetCertificate(ctx context.Context, config DigiCertCA) (*DigiCertCertificate, error)
}
