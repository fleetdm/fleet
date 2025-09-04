package fleet

import (
	"context"
)

type HydrantCertificate struct {
	Certificate []byte
}

type HydrantService interface {
	// ValidateHydrantURL validates that the provided URL in the HydrantCA is reachable via the
	// /cacerts endpoint. It is not responsible for checking the credentials.
	ValidateHydrantURL(ctx context.Context, hydrantCA HydrantCA) error
	// GetCertificate retrieves a certificate from the Hydrant CA using the provided CSR which must
	// be in base64 encoded PKCS#10 format(i.e. PEM format without the header, footer or newlines).
	// The CSR format must match the template configured on the Hydrant server. The certificate is
	// returned in a similar format as the CSR(i.e. base64 encoded PKCS#7)
	GetCertificate(ctx context.Context, hydrantCA HydrantCA, csr string) (*HydrantCertificate, error)
}
