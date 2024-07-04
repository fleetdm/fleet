package mdmcrypto

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (s *SCEPVerifier) Verify(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("no certificate provided")
	}

	opts := x509.VerifyOptions{
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Roots:     x509.NewCertPool(),
	}

	// TODO(roberto): nano interfaces don't allow to pass a context to this function
	assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	})
	if err != nil {
		return fmt.Errorf("loading existing assets from the database: %w", err)
	}

	if ok := opts.Roots.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value); !ok {
		return errors.New("unable to append cerver SCEP cert to pool verifier")
	}

	if _, err := cert.Verify(opts); err != nil {
		return err
	}

	return nil
}
