package hostidentity

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
)

func initAssets(ds fleet.Datastore) error {
	// Check if we have existing certs and keys
	expectedAssets := []fleet.MDMAssetName{
		fleet.MDMAssetHostIdentityCACert,
		fleet.MDMAssetHostIdentityCAKey,
	}
	savedAssets, err := ds.GetAllMDMConfigAssetsByName(context.Background(), expectedAssets, nil)
	if err != nil {
		// allow not found errors as it means we're generating the assets for the first time.
		if !fleet.IsNotFound(err) {
			return fmt.Errorf("loading existing host identity assets from the database: %w", err)
		}
	}

	if len(savedAssets) != len(expectedAssets) {
		// Then we should create them
		caCert := depot.NewCACert(
			depot.WithYears(10),
			depot.WithCommonName("Fleet Host Identity CA"),
			// Signal that the CA is local to the deployment and not necessarily managed by Fleet or another external vendor
			depot.WithOrganization("Local Certificate Authority"),
		)
		scepCert, scepKey, err := depot.NewCACertKey(caCert)
		if err != nil {
			return fmt.Errorf("generating host identity SCEP cert and key: %w", err)
		}

		// Store our config assets encrypted
		var assets []fleet.MDMConfigAsset
		for k, v := range map[fleet.MDMAssetName][]byte{
			fleet.MDMAssetHostIdentityCACert: certificate.EncodeCertPEM(scepCert),
			fleet.MDMAssetHostIdentityCAKey:  certificate.EncodePrivateKeyPEM(scepKey),
		} {
			assets = append(assets, fleet.MDMConfigAsset{
				Name:  k,
				Value: v,
			})
		}

		if err := ds.InsertMDMConfigAssets(context.Background(), assets, nil); err != nil {
			return fmt.Errorf("inserting host identity SCEP assets: %w", err)
		}
	}
	return nil
}
