package fleet

import (
	"context"
)

type HydrantCertificate struct {
	Certificate []byte
}

type HydrantService interface {
	ValidateHydrantURL(ctx context.Context, hydrantCA HydrantCA) error
	GetCertificate(ctx context.Context, hydrantCA HydrantCA, csr string) (*HydrantCertificate, error)
}
