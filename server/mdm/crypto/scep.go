package mdmcrypto

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"
)

var _ mdm.CertVerifier = (*SCEPVerifier)(nil)

type SCEPVerifier struct {
	ds fleet.MDMAssetRetriever
}

func NewSCEPVerifier(ds fleet.MDMAssetRetriever) *SCEPVerifier {
	return &SCEPVerifier{
		ds: ds,
	}
}

func (s *SCEPVerifier) Verify(ctx context.Context, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("no certificate provided")
	}

	opts := x509.VerifyOptions{
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Roots:     x509.NewCertPool(),
	}

	rootCert, err := assets.X509Cert(ctx, s.ds, fleet.MDMAssetCACert)
	if err != nil {
		return fmt.Errorf("loading existing assets from the database: %w", err)
	}
	opts.Roots.AddCert(rootCert)

	// the default SCEP cert issued by fleet doesn't have any extra key
	// usages, however, customers might configure the server with any
	// certificate they want (generally for touchless MDM migrations)
	//
	// given that go verifies ext key usages on the whole chain, we relax
	// the constraints when the provided certificate has any ext key usage
	// that would cause a failure.
	if hasOtherKeyUsages(rootCert, x509.ExtKeyUsageClientAuth) {
		opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	}

	if _, err := cert.Verify(opts); err != nil {
		return err
	}

	return nil
}

func hasOtherKeyUsages(cert *x509.Certificate, usage x509.ExtKeyUsage) bool {
	for _, u := range cert.ExtKeyUsage {
		if u != usage {
			return true
		}
	}
	return false
}
