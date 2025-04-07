package certverify

import (
	"context"
	"crypto/x509"
	"errors"
)

// PoolVerifier is a simple certificate verifier
type PoolVerifier struct {
	verifyOpts x509.VerifyOptions
}

// NewPoolVerifier creates a new Verifier
func NewPoolVerifier(rootsPEM []byte, keyUsages ...x509.ExtKeyUsage) (*PoolVerifier, error) {
	opts := x509.VerifyOptions{
		KeyUsages: keyUsages,
		Roots:     x509.NewCertPool(),
	}
	if len(rootsPEM) == 0 || !opts.Roots.AppendCertsFromPEM(rootsPEM) {
		return nil, errors.New("could not append root CA(s)")
	}
	return &PoolVerifier{
		verifyOpts: opts,
	}, nil
}

// Verify performs certificate verification
func (v *PoolVerifier) Verify(_ context.Context, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("missing MDM certificate")
	}
	if _, err := cert.Verify(v.verifyOpts); err != nil {
		return err
	}
	return nil
}
