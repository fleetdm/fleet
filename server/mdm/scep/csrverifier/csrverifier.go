// Package csrverifier defines an interface for CSR verification.
package csrverifier

import (
	"crypto/x509"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
)

// CSRVerifier verifies the raw decrypted CSR.
type CSRVerifier interface {
	Verify(data []byte) (bool, error)
}

// Middleware wraps next in a CSRSigner that runs verifier
func Middleware(verifier CSRVerifier, next scepserver.CSRSigner) scepserver.CSRSignerFunc {
	return func(m *scep.CSRReqMessage) (*x509.Certificate, error) {
		ok, err := verifier.Verify(m.RawDecrypted)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("CSR verify failed")
		}
		return next.SignCSR(m)
	}
}
