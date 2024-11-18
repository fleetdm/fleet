package certverify

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
)

// SignatureVerifier is a simple certificate verifier
type SignatureVerifier struct {
	ca *x509.Certificate
}

// NewSignatureVerifier creates a new Verifier
func NewSignatureVerifier(rootPEM []byte) (*SignatureVerifier, error) {
	ca, err := cryptoutil.DecodePEMCertificate(rootPEM)
	if err != nil {
		return nil, err
	}
	if ca == nil {
		return nil, errors.New("nil PEM certificate")
	}
	return &SignatureVerifier{ca: ca}, nil
}

// Verify checks only the signature of the certificate against the CA
func (v *SignatureVerifier) Verify(_ context.Context, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("missing MDM certificate")
	}
	return cert.CheckSignatureFrom(v.ca)
}
