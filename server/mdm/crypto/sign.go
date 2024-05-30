package mdmcrypto

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/micromdm/micromdm/pkg/crypto/profileutil"
)

// Sign signs an enrollment profile using a certificate from the datastore
func Sign(ctx context.Context, profile []byte, ds fleet.MDMAssetRetriever) ([]byte, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
		fleet.MDMAssetCAKey,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading SCEP keypair from the database")
	}
	cert, err := tls.X509KeyPair(assets[fleet.MDMAssetCACert].Value, assets[fleet.MDMAssetCAKey].Value)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing SCEP keypair")
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing SCEP certificate")
	}

	signed, err := profileutil.Sign(cert.PrivateKey, leaf, profile)
	if err != nil {
		return nil, fmt.Errorf("signing profile with the specified key: %w", err)
	}

	return signed, nil
}
