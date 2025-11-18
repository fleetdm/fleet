package fleet

import (
	"context"
)

type ESTCertificate struct {
	Certificate []byte
}

type ESTService interface {
	// ValidateESTURL validates that the provided URL in the ESTProxyCA is reachable via the
	// /cacerts endpoint. It is not responsible for checking the credentials.
	ValidateESTURL(ctx context.Context, estProxyCA ESTProxyCA) error
	// GetCertificate retrieves a certificate from the EST CA using the provided CSR which must
	// be in base64 encoded PKCS#10 format(i.e. PEM format without the header, footer or newlines).
	// The CSR format must match the template configured on the EST server. The certificate is
	// returned in a similar format as the CSR(i.e. base64 encoded PKCS#7)
	GetCertificate(ctx context.Context, estProxyCA ESTProxyCA, csr string) (*ESTCertificate, error)
}
