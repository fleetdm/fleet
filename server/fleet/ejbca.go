package fleet

import (
	"context"
	"time"
)

// EJBCACertificate is the result of a successful certificate enrollment against
// an EJBCA REST API. Fleet generates the keypair locally, sends the CSR, then
// wraps the issued cert and its key in a PKCS#12 for delivery to MDM hosts.
type EJBCACertificate struct {
	PfxData        []byte
	Password       string
	NotValidBefore time.Time
	NotValidAfter  time.Time
	SerialNumber   string
}

// EJBCAService is the contract for the EJBCA REST API client. Implementation
// lives in ee/server/service/ejbca/.
type EJBCAService interface {
	// VerifyConnection probes GET /v1/ca/status over mTLS to confirm the CA
	// configuration (URL + client cert + trust bundle) reaches EJBCA and
	// authenticates successfully.
	VerifyConnection(ctx context.Context, config EJBCACA) error
	// GetCertificate enrolls a certificate via POST /v1/certificate/pkcs10enroll
	// using the supplied EJBCACA configuration. Caller is responsible for
	// expanding any Fleet variables in config fields before calling.
	GetCertificate(ctx context.Context, config EJBCACA) (*EJBCACertificate, error)
}
